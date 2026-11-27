// Запуск: node review.js diff.txt > review.json
const fs = require('fs');

const diff = fs.readFileSync(process.argv[2], 'utf8');
const rules = fs.readFileSync('.github/review-instructions.md', 'utf8');

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

const prompt = `Сделай код-ревью этого diff строго по правилам ниже.
Комментируй только то, что нарушает правила. Уровень (severity) бери из правил.

<rules>
${rules}
</rules>

В diff перед каждой строкой стоит её номер в новой версии файла, например "42: +код".
В поле "line" указывай именно этот номер. Удалённые строки (без номера) не комментируй.

Ответь только JSON без markdown:
{"summary": "общий вывод", "comments": [{"file": "путь из строки +++ b/...", "line": номер строки, "severity": "error|warning|nit", "message": "что не так", "suggestion": "исправленный код (необязательно)"}]}

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
      messages: [{ role: 'user', content: prompt }],
    }),
  });

  const data = await res.json();
  const text = data.choices[0].message.content;

  // Берём JSON из ответа модели и печатаем его
  const json = text.slice(text.indexOf('{'), text.lastIndexOf('}') + 1);
  console.log(json);
}

main();