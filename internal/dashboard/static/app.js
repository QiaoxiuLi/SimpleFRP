const byId = (id) => document.getElementById(id);
const rows = new Map();

function bytes(value) {
  let n = Math.max(0, Number(value) || 0);
  const units = ["B", "KiB", "MiB", "GiB", "TiB"];
  let unit = 0;
  while (n >= 1024 && unit < units.length - 1) { n /= 1024; unit++; }
  return `${n.toFixed(unit ? 1 : 0)} ${units[unit]}`;
}

function render(data) {
  byId("heartbeat-port").textContent = data.heartbeat_port || "Not available";
  const seen = new Set();
  for (const tunnel of data.tunnels || []) {
    seen.add(tunnel.id);
    let row = rows.get(tunnel.id);
    if (!row) {
      row = document.createElement("tr");
      row.dataset.tunnelId = tunnel.id;
      for (let i = 0; i < 6; i++) row.appendChild(document.createElement("td"));
      rows.set(tunnel.id, row);
    }
    const values = [tunnel.id, tunnel.public_port || "Not assigned", tunnel.local_port || "Not assigned",
      `${bytes(tunnel.bytes_per_second)}/s`, bytes(tunnel.total_bytes), tunnel.status];
    values.forEach((value, i) => { row.cells[i].textContent = String(value); });
    row.dataset.state = tunnel.status;
    byId("tunnels").appendChild(row);
  }
  for (const [id, row] of rows) {
    if (!seen.has(id)) { row.remove(); rows.delete(id); }
  }
  byId("empty-state").hidden = seen.size !== 0;
  byId("connection-state").textContent = data.online ? "Live" : "Offline";
  byId("connection-state").dataset.state = data.online ? "live" : "offline";
  byId("updated-at").textContent = `Updated ${new Date().toLocaleTimeString()}`;
}

async function refresh() {
  if (document.hidden) { setTimeout(refresh, 1000); return; }
  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), 5000);
  try {
    const response = await fetch("/api/status", { cache: "no-store", signal: controller.signal });
    if (!response.ok) throw new Error("Status unavailable");
    render(await response.json());
  } catch {
    byId("connection-state").textContent = "Unavailable";
    byId("connection-state").dataset.state = "offline";
    for (const row of rows.values()) { row.cells[3].textContent = "0 B/s"; row.cells[5].textContent = "unknown"; }
  } finally {
    clearTimeout(timeout);
    setTimeout(refresh, 1000);
  }
}

refresh();
