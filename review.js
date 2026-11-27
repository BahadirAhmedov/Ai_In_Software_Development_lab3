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

// Проставляем номера строк новой версии файла, чтобы модели не нужно было считать.
// Заодно запоминаем, какие строки каких файлов есть в diff (на них можно комментировать).
const allowed = {}; // { "app/main.go": Set(12, 13, ...) }

function numberDiff(text) {
  let file = null;
  let line = 0;
  return text.split('\n').map(row => {
    if (row.startsWith('+++ ')) {
      file = row.startsWith('+++ b/') ? row.slice(6) : null;
      if (file) allowed[file] = new Set();
      return row;
    }
    const hunk = row.match(/^@@ -\d+(?:,\d+)? \+(\d+)/);
    if (hunk) { line = Number(hunk[1]); return row; }
    if (row.startsWith('---')) return row;
    if (row.startsWith('+') || row.startsWith(' ')) {
      if (file) allowed[file].add(line);
      return `${line++}: ${row}`;
    }
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

// Что публикуем, если что-то пошло не так. Workflow при этом не падает.
function fallback(reason) {
  console.error('AI-ревью не выполнено:', reason); // подробности — в лог Actions
  console.log(JSON.stringify({
    summary: `AI-ревью не выполнено: ${reason}. Нужна ручная проверка PR.`,
    comments: [],
  }));
}

// Достаём JSON из ответа и проверяем, что он нужной формы
function parseReview(text) {
  const start = text.indexOf('{');
  const end = text.lastIndexOf('}');
  if (start === -1 || end === -1) throw new Error('в ответе модели нет JSON');

  let review;
  try {
    review = JSON.parse(text.slice(start, end + 1));
  } catch {
    throw new Error('модель вернула невалидный JSON');
  }

  if (typeof review !== 'object' || review === null || !Array.isArray(review.comments)) {
    throw new Error('в ответе модели нет списка comments');
  }

  // Плохие отдельные комментарии выбрасываем, хорошие оставляем
  const comments = review.comments.filter(c =>
    c && typeof c.message === 'string' &&
    allowed[c.file] && allowed[c.file].has(Number(c.line))
  ).map(c => ({
    file: c.file,
    line: Number(c.line),
    severity: String(c.severity || 'minor'),
    message: c.message,
    ...(typeof c.suggestion === 'string' && c.suggestion ? { suggestion: c.suggestion } : {}),
  }));

  const dropped = review.comments.length - comments.length;
  let summary = typeof review.summary === 'string' ? review.summary : '';
  if (dropped > 0) {
    console.error(`Отброшено некорректных комментариев: ${dropped}`);
    summary += `\n\n_Отброшено комментариев с некорректным файлом или строкой: ${dropped}._`;
  }
  return { summary, comments };
}

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

  if (!res.ok) {
    console.error(await res.text());
    return fallback(`API модели вернул ошибку ${res.status}`);
  }

  const data = await res.json();
  const text = data?.choices?.[0]?.message?.content;
  if (!text) return fallback('модель вернула пустой ответ');

  try {
    console.log(JSON.stringify(parseReview(text)));
  } catch (err) {
    console.error('Ответ модели:\n' + text);
    fallback(err.message);
  }
}

// Любая неожиданная ошибка (сеть, таймаут и т.п.) — тоже fallback, а не падение
main().catch(err => fallback(`ошибка при обращении к модели (${err.message})`));