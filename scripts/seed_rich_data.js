const http = require('http');

const queries = [
  // Fast SLM / Cacheable queries (Tier 1 & Tier 0)
  {
    key: "cp-live-sup-3b714c",
    dept: "Customer Support",
    prompt: "Categorize the ticket urgency into Low, Medium, High: 'Customer forgot password and cannot log into mobile banking app.'",
  },
  {
    key: "cp-live-sup-3b714c",
    dept: "Customer Support",
    prompt: "Categorize the ticket urgency into Low, Medium, High: 'Customer forgot password and cannot log into mobile banking app.'", // Cache hit!
  },
  {
    key: "cp-live-mkt-5c920e",
    dept: "Marketing",
    prompt: "Extract the company name, job role, and contact email from this bio: 'Alex Rivera, VP of Growth at CloudScale AI. Reach me at alex@cloudscale.ai.'",
  },
  {
    key: "cp-live-mkt-5c920e",
    dept: "Marketing",
    prompt: "Draft 3 catchy, professional email subject lines announcing our Q3 feature release for enterprise data warehousing.",
  },
  {
    key: "cp-live-eng-9f821a",
    dept: "Engineering",
    prompt: "Convert this timestamp 1731713200 to ISO 8601 UTC string format and explain the format.",
  },
  {
    key: "cp-live-eng-9f821a",
    dept: "Engineering",
    prompt: "Write a high-performance regex pattern to validate strict E.164 international telephone numbers.",
  },
  {
    key: "cp-live-eng-9f821a",
    dept: "Engineering",
    prompt: "Write a high-performance regex pattern to validate strict E.164 international telephone numbers.", // Cache hit!
  },
  // Moderate queries (Tier 2 candidate / escalation)
  {
    key: "cp-live-eng-9f821a",
    dept: "Engineering",
    prompt: "Refactor this TypeScript async debounce function to support immediate execution on leading edge and cancellation token: function debounce(fn, wait) { let t; return (...args) => { clearTimeout(t); t = setTimeout(() => fn(...args), wait); }; }",
  },
  {
    key: "cp-live-gen-default",
    dept: "General",
    prompt: "Explain the algorithmic differences between B-Tree and LSM-Tree storage engines in RocksDB versus PostgreSQL, focusing on write amplification and random read throughput.",
  },
  {
    key: "cp-live-sup-3b714c",
    dept: "Customer Support",
    prompt: "Analyze the tone and subtext of this escalated customer complaint: 'We have experienced 4 hours of unannounced downtime in our production environment during peak trading hours. This breach of our 99.99% SLA is completely unacceptable.'",
  },
  // Frontier / Complex queries (Tier 3 direct or escalated)
  {
    key: "cp-live-eng-9f821a",
    dept: "Engineering",
    prompt: "Formal reasoning problem: Step by step, prove the correctness of Paxos consensus under non-Byzantine asynchronous network partitions. Address the commit phase invariant, quorum intersection theorem, and leader ballot monotonicity.",
  },
  {
    key: "cp-live-eng-9f821a",
    dept: "Engineering",
    prompt: "Audit this C memory allocator snippet for potential double-free and use-after-free race conditions under multithreaded jemalloc heap contention: void free_chunk(arena_t *a, chunk_t *c) { atomic_sub(&a->allocs, 1); if (c->magic == 0xDEAD) abort(); c->next = a->freelist; a->freelist = c; }",
  },
  {
    key: "cp-live-mkt-5c920e",
    dept: "Marketing",
    prompt: "Synthesize an executive multi-tier pricing strategy for a B2B generative AI infrastructure platform. Model cohort retention, gross margin expansion, token consumption elasticity, and enterprise minimum commitments.",
  },
  {
    key: "cp-live-gen-default",
    dept: "General",
    prompt: "Derive the mathematical loss gradient for multi-query attention (MQA) versus grouped-query attention (GQA) with respect to key-value projection weights during backpropagation.",
  },
  {
    key: "cp-live-sup-3b714c",
    dept: "Customer Support",
    prompt: "Draft an enterprise incident post-mortem according to SRE best practices for an outage caused by TLS certificate expiration on the internal ingress API gateway.",
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
          const parsed = JSON.parse(body);
          resolve(parsed);
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
  console.log(`Seeding ${queries.length} realistic enterprise queries...`);
  for (let i = 0; i < queries.length; i++) {
    const q = queries[i];
    try {
      const res = await sendQuery(q);
      console.log(`[${i + 1}/${queries.length}] (${q.dept}) Prompt: "${q.prompt.slice(0, 40)}..." -> ID: ${res.id || 'err'}`);
      await new Promise(r => setTimeout(r, 200));
    } catch (e) {
      console.error(`Error sending query ${i}:`, e.message);
    }
  }
  console.log('Seeding completed successfully!');
}

run();
