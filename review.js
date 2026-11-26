// Запуск: node review.js diff.txt > review.json
const fs = require('fs');

const diff = fs.readFileSync(process.argv[2], 'utf8');

const prompt = `Сделай код-ревью этого diff. Ищи баги и уязвимости, не стиль.
Ответь только JSON без markdown:
{"summary": "общий вывод", "comments": [{"file": "путь", "line": номер строки в новой версии, "severity": "critical|warning|nit", "message": "что не так", "suggestion": "исправленный код (необязательно)"}]}

<diff>
${diff}
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