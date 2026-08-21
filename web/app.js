const tenantInput = document.querySelector("#tenant");
const rows = document.querySelector("#rows");
const healthDot = document.querySelector("#health-dot");
const healthLabel = document.querySelector("#health-label");

async function load() {
  const tenant = tenantInput.value.trim();
  try {
    const [health, response] = await Promise.all([
      fetch("/healthz"),
      fetch("/api/v1/notifications?limit=100", { headers: { "X-Tenant-ID": tenant } }),
    ]);
    if (!health.ok || !response.ok) throw new Error("API unavailable");
    const payload = await response.json();
    const items = payload.items || [];
    const counts = { queued: 0, processing: 0, delivered: 0, failed: 0 };
    for (const item of items) if (counts[item.status] !== undefined) counts[item.status]++;
    for (const [key, value] of Object.entries(counts)) document.querySelector(`#${key}`).textContent = value;
    rows.innerHTML = items.length ? items.map((item) => `<tr><td>${item.id}</td><td>${item.channel}</td><td>${item.status}</td><td>${item.provider || "-"}</td><td>${new Date(item.updated_at).toLocaleString()}</td></tr>`).join("") : '<tr><td colspan="5" class="empty">No notifications for this tenant</td></tr>';
    healthDot.className = "ok";
    healthLabel.textContent = "API healthy";
    document.querySelector("#updated").textContent = `Updated ${new Date().toLocaleTimeString()}`;
  } catch (error) {
    healthDot.className = "bad";
    healthLabel.textContent = error.message;
  }
}

document.querySelector("#refresh").addEventListener("click", load);
load();
