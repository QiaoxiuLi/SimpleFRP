const byId = (id) => document.getElementById(id);

function bytes(n) {
  if (!n) return "0 B";
  const units = ["B", "KB", "MB", "GB", "TB"];
  let i = 0;
  while (n >= 1024 && i < units.length - 1) { n /= 1024; i++; }
  return `${n.toFixed(i ? 1 : 0)} ${units[i]}`;
}

async function refreshStatus() {
  const res = await fetch("/api/status");
  const data = await res.json();
  byId("clients").textContent = data.clients;
  byId("tunnels").textContent = data.active_tunnels;
  byId("traffic").textContent = bytes(data.traffic.total_bytes);
}

async function refreshConnections() {
  const res = await fetch("/api/connections?limit=100");
  const rows = await res.json();
  const tbody = byId("connections");
  for (const row of rows) {
    let tr = document.querySelector(`[data-connection-id="${row.id}"]`);
    if (!tr) {
      tr = document.createElement("tr");
      tr.dataset.connectionId = row.id;
      tbody.prepend(tr);
    }
    tr.innerHTML = `<td>${row.ipv4 || ""}</td><td>${row.ipv6 || ""}</td><td>${row.connected_at || ""}</td><td>${row.duration || ""}</td><td>${row.tunnel_id}</td><td>${row.local_port}</td><td>${row.public_port}</td><td>${row.status}</td>`;
  }
}

document.querySelectorAll("[data-range]").forEach((button) => {
  button.addEventListener("click", async () => {
    const res = await fetch(`/api/traffic?range=${button.dataset.range}`);
    const data = await res.json();
    byId("traffic").textContent = bytes(data.total_bytes);
  });
});

const events = new EventSource("/events");
events.onmessage = () => refreshStatus();
events.addEventListener("traffic_updated", refreshStatus);
events.addEventListener("client_connected", () => { refreshStatus(); refreshConnections(); });

refreshStatus();
refreshConnections();
