// Запуск: node review.js diff.txt > review.json
const fs = require('fs');

const diff = fs.readFileSync(process.argv[2], 'utf8');
const rules = fs.readFileSync('.github/review-instructions.md', 'utf8');

// Порог размера PR: больше — модель не зовём
const MAX_LINES = 500; // изменённых строк (+ и -)
const MAX_FILES = 20;  // изменённых файлов

const rows = diff.split('\n');
const changedLines = rows.filter(r =>
  (r.startsWith('+') || r.startsWith('-')) && !r.startsWith('+++') && !r.startsWith('---')
).length;
const changedFiles = rows.filter(r => r.startsWith('diff --git')).length;

if (changedLines > MAX_LINES || changedFiles > MAX_FILES) {
  console.log(JSON.stringify({
    summary: `PR слишком большой для автоматического ревью (изменено строк: ${changedLines}, файлов: ${changedFiles}; ` +
      `лимит: ${MAX_LINES} строк, ${MAX_FILES} файлов), review пропущен.`,
    comments: [],
  }));
  process.exit(0);
}

// Проставляем номера строк новой версии файла, чтобы модели не нужно было считать
function numberDiff(text) {
  let line = 0;
  return text.split('\n').map(row => {
    const hunk = row.match(/^@@ -\d+(?:,\d+)? \+(\d+)/);
    if (hunk) { line = Number(hunk[1]); return row; }
    if (row.startsWith('+++') || row.startsWith('---')) return row;
    if (row.startsWith('+') || row.startsWith(' ')) return `${line++}: ${row}`;
    if (row.startsWith('-')) return `    ${row}`; // удалённая строка — без номера
    return row;
  }).join('\n');
}

// Всё о том, что проверять, уровнях и формате ответа — в review-instructions.md.
// Здесь только пояснение к нумерации строк, чтобы не было двух источников правил.
const prompt = `Сделай ревью этого diff по инструкциям.

Перед каждой строкой diff стоит её номер в новой версии файла, например "42: +код".
В поле "line" указывай именно этот номер. Строки без номера (удалённые) не комментируй.
Путь файла бери из строки "+++ b/<путь>".

<diff>
${numberDiff(diff)}
</diff>`;

async function main() {
  const res = await fetch('https://api.z.ai/api/paas/v4/chat/completions', {
    method: 'POST',
    headers: {
      'content-type': 'application/json',
      'authorization': `Bearer ${process.env.GLM_API_KEY}`,
    },
    body: JSON.stringify({
      model: 'glm-5.3',
      max_tokens: 8192,
      messages: [
        { role: 'system', content: rules },
        { role: 'user', content: prompt },
      ],
    }),
  });

  const data = await res.json();
  const text = data.choices[0].message.content;

  // Берём JSON из ответа модели и печатаем его
  const json = text.slice(text.indexOf('{'), text.lastIndexOf('}') + 1);
  console.log(json);
}

main();