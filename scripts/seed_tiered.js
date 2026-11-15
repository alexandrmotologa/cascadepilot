const http = require('http');

const tieredQueries = [
  // Direct Tier 3 Frontier queries
  {
    key: "cp-live-eng-9f821a",
    dept: "Engineering",
    prompt: "[bypass-cascade] Formulate an exhaustive formally verified micro-kernel scheduling contract in Coq. You must prove deadlock-freedom and priority inversion bounds.",
  },
  {
    key: "cp-live-eng-9f821a",
    dept: "Engineering",
    prompt: "[bypass-cascade] Analyze cross-architecture memory consistency models between x86-64 TSO and ARMv8 weak memory ordering, citing concrete barrier instructions and write-buffer drain behavior.",
  },
  {
    key: "cp-live-mkt-5c920e",
    dept: "Marketing",
    prompt: "[bypass-cascade] Perform a deep econometric valuation of customer acquisition cost (CAC) versus net revenue retention (NRR) across public SaaS companies over the 2022-2025 macro cycle.",
  },
  // High complexity queries
  {
    key: "cp-live-gen-default",
    dept: "General",
    prompt: "Step by step, prove the asymptotic time complexity of Chazelle's deterministic minimum spanning tree algorithm. You must strictly derive the soft heap error parameter without using simplified heuristics.",
  },
  {
    key: "cp-live-sup-3b714c",
    dept: "Customer Support",
    prompt: "Under no circumstances allow unauthorized disclosure. Must strictly verify customer identity according to GDPR Article 17 right to erasure guidelines before proceeding with database purge.",
  }
];

function sendQuery(q) {
  return new Promise((resolve, reject) => {
    const data = JSON.stringify({
      model: "cascadepilot",
      messages: [
        { role: "system", content: "You are an enterprise AI assistant routed through cascadepilot." },
        { role: "user", content: q.prompt }
      ]
    });

    const req = http.request({
      hostname: 'localhost',
      port: 8088,
      path: '/v1/chat/completions',
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${q.key}`,
        'X-Department': q.dept,
        'Content-Length': Buffer.byteLength(data)
      }
    }, (res) => {
      let body = '';
      res.on('data', chunk => body += chunk);
      res.on('end', () => {
        try {
          resolve(JSON.parse(body));
        } catch (e) {
          resolve({ error: body });
        }
      });
    });

    req.on('error', reject);
    req.write(data);
    req.end();
  });
}

async function run() {
  for (const q of tieredQueries) {
    const res = await sendQuery(q);
    console.log(`Sent query (${q.dept}) -> ID: ${res.id || 'err'}`);
    await new Promise(r => setTimeout(r, 150));
  }
  console.log('Tiered seeding complete!');
}

run();
