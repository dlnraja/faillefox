// Faillefox — logique du panneau de contrôle.
// Vanilla JS, aucune dépendance. Communique avec l'API loopback du pare-feu.

const API = ""; // même origine (servi par le démon Go)

// ---- thème (clair/sombre/auto) — appliqué le plus tôt possible -----------
// Ordre de priorité : localStorage > thème système. Bascule au clic du bouton.
const THEMES = ["dark", "light", "auto"]; // ordre de bascule cyclique
const THEME_ICONS = { dark: "🌙", light: "☀️", auto: "🖥️" };

function applyTheme(theme) {
  let effective = theme;
  if (theme === "auto") {
    effective = window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light";
  }
  document.documentElement.setAttribute("data-theme", effective);
  const btn = document.getElementById("theme-toggle");
  if (btn) btn.textContent = THEME_ICONS[theme] || "🌙";
}

function currentTheme() {
  return localStorage.getItem("faillefox-theme") || "dark";
}

function cycleTheme() {
  const cur = currentTheme();
  const idx = THEMES.indexOf(cur);
  const next = THEMES[(idx + 1) % THEMES.length];
  localStorage.setItem("faillefox-theme", next);
  applyTheme(next);
}

// Applique le thème immédiatement (évite le flash de couleur au chargement).
applyTheme(currentTheme());
// Si en mode auto, on suit les changements de préférence système en direct.
window.matchMedia("(prefers-color-scheme: dark)").addEventListener("change", () => {
  if (currentTheme() === "auto") applyTheme("auto");
});

document.addEventListener("DOMContentLoaded", () => {
  const btn = document.getElementById("theme-toggle");
  if (btn) btn.addEventListener("click", cycleTheme);
  applyTheme(currentTheme()); // met à jour l'icône du bouton
});

// ---- utilitaires ----------------------------------------------------------
async function getJSON(path) {
  const r = await fetch(API + path);
  if (!r.ok) throw new Error(`${path}: ${r.status}`);
  return r.json();
}
async function postJSON(path, body) {
  const r = await fetch(API + path, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body || {}),
  });
  if (!r.ok && r.status !== 204) throw new Error(`${path}: ${r.status}`);
  return r.status === 204 ? null : r.json();
}
function esc(s) {
  return String(s ?? "").replace(/[&<>"']/g, (c) => ({
    "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;",
  }[c]));
}
function fmtTime(iso) {
  const d = new Date(iso);
  return d.toLocaleTimeString("fr-FR");
}

// ---- navigation par onglets ----------------------------------------------
document.querySelectorAll(".tab").forEach((tab) => {
  tab.addEventListener("click", () => {
    document.querySelectorAll(".tab").forEach((t) => t.classList.remove("active"));
    document.querySelectorAll(".panel").forEach((p) => p.classList.remove("active"));
    tab.classList.add("active");
    document.getElementById("tab-" + tab.dataset.tab).classList.add("active");
  });
});

// ---- état global ----------------------------------------------------------
let apps = [];
let rules = [];

// ---- applications (mode simple) ------------------------------------------
async function loadApps() {
  apps = await getJSON("/api/apps");
  const ul = document.getElementById("apps-list");
  ul.innerHTML = apps
    .map(
      (a) => `
      <li>
        <div>
          <div class="app-name">${esc(a.name)}</div>
          <div class="app-path">${esc(a.path)}</div>
        </div>
        <label class="switch">
          <input type="checkbox" data-app="${esc(a.id)}" class="app-toggle" />
          <span class="slider"></span>
        </label>
      </li>`
    )
    .join("");

  // Remplit aussi le sélecteur du mode avancé.
  const sel = document.getElementById("rule-app");
  sel.innerHTML =
    `<option value="">(toutes les applications)</option>` +
    apps.map((a) => `<option value="${esc(a.id)}">${esc(a.name)}</option>`).join("");

  // Synchronise l'état des interrupteurs avec les règles existantes.
  syncAppToggles();
}

function syncAppToggles() {
  document.querySelectorAll(".app-toggle").forEach((cb) => {
    const denyRule = rules.find(
      (r) => r.app_id === cb.dataset.app && r.action === "deny" && !r.port && !r.ip
    );
    cb.checked = !denyRule; // coché = Internet autorisé
  });
}

// Gestion du clic sur un interrupteur : crée/supprime une règle deny.
document.getElementById("apps-list").addEventListener("change", async (e) => {
  if (!e.target.classList.contains("app-toggle")) return;
  const appId = e.target.dataset.app;
  const allow = e.target.checked;
  try {
    if (allow) {
      // Retire une éventuelle règle deny existante.
      const deny = rules.find(
        (r) => r.app_id === appId && r.action === "deny" && !r.port && !r.ip
      );
      if (deny) await fetch(`${API}/api/rules?id=${encodeURIComponent(deny.id)}`, { method: "DELETE" });
    } else {
      await postJSON("/api/rules", {
        app_id: appId,
        action: "deny",
        protocol: "",
        port: 0,
        ip: "",
        note: "bloqué depuis le mode simple",
      });
    }
    await refreshRules();
  } catch (err) {
    alert("Erreur: " + err.message);
    e.target.checked = !allow; // rollback visuel
  }
});

// ---- politique par défaut ------------------------------------------------
function renderDefault(d) {
  document.querySelectorAll("#default-choice button").forEach((b) => {
    b.classList.toggle("active", b.dataset.decision === d);
  });
}
document.getElementById("default-choice").addEventListener("click", async (e) => {
  const btn = e.target.closest("button");
  if (!btn) return;
  try {
    await postJSON("/api/default", { decision: btn.dataset.decision });
    renderDefault(btn.dataset.decision);
    refreshStatus();
  } catch (err) {
    alert("Erreur: " + err.message);
  }
});

// ---- règles avancées ------------------------------------------------------
async function refreshRules() {
  rules = await getJSON("/api/rules");
  const body = document.getElementById("rules-body");
  body.innerHTML = rules
    .map(
      (r) => `
      <tr>
        <td>${esc(appName(r.app_id))}</td>
        <td><span class="badge ${r.action}">${r.action}</span></td>
        <td>${esc(r.protocol || "tous")}</td>
        <td>${r.port || "tous"}</td>
        <td>${esc(r.ip || "toutes")}</td>
        <td><button class="del-btn" data-id="${esc(r.id)}" title="Supprimer">✕</button></td>
      </tr>`
    )
    .join("");
  syncAppToggles();
}
function appName(id) {
  if (!id) return "(toutes)";
  const a = apps.find((x) => x.id === id);
  return a ? a.name : id;
}

document.getElementById("rules-body").addEventListener("click", async (e) => {
  if (!e.target.classList.contains("del-btn")) return;
  if (!confirm("Supprimer cette règle ?")) return;
  await fetch(`${API}/api/rules?id=${encodeURIComponent(e.target.dataset.id)}`, { method: "DELETE" });
  refreshRules();
  refreshStatus();
});

document.getElementById("rule-form").addEventListener("submit", async (e) => {
  e.preventDefault();
  const fd = new FormData(e.target);
  const port = parseInt(fd.get("port"), 10) || 0;
  await postJSON("/api/rules", {
    app_id: fd.get("app_id"),
    action: fd.get("action"),
    protocol: fd.get("protocol"),
    port,
    ip: fd.get("ip"),
  });
  e.target.reset();
  refreshRules();
  refreshStatus();
});

// ---- statut ---------------------------------------------------------------
async function refreshStatus() {
  const s = await getJSON("/api/status");
  const dot = document.getElementById("status-dot");
  const txt = document.getElementById("status-text");
  txt.textContent = `Pilote: ${s.driver} · ${s.rules_count} règle(s) · défaut: ${s.default_decision}`;
  dot.className = "dot ok";
  renderDefault(s.default_decision);
}

// ---- journal temps réel (SSE) --------------------------------------------
function startEventStream() {
  const es = new EventSource(API + "/api/events");
  const logs = document.getElementById("logs");
  es.onmessage = (msg) => {
    const ev = JSON.parse(msg.data);
    const line = document.createElement("div");
    line.className = "log-line";
    line.innerHTML = `
      <span class="log-time">${esc(fmtTime(ev.At))}</span>
      <span class="log-app">${esc(ev.Connection.app_name)}</span>
      <span class="log-target">${esc(ev.Connection.protocol)} → ${esc(ev.Connection.remote_addr)}:${ev.Connection.remote_port}</span>
      <span class="log-verdict ${ev.Decision}">${ev.Decision}</span>`;
    logs.prepend(line);
    // Limite à 300 lignes pour éviter l'explosion mémoire.
    while (logs.children.length > 300) logs.removeChild(logs.lastChild);
  };
  es.onerror = () => {
    document.getElementById("status-dot").className = "dot bad";
  };
}

// ---- auto-update : état du rafraîchissement des listes -------------------
async function refreshUpdater() {
  const el = document.getElementById("update-status");
  try {
    const data = await getJSON("/api/updater");
    if (!data.enabled) {
      el.innerHTML = `<span class="warn">Auto-update désactivé</span> — relancez sans <code>-no-autoupdate</code>.`;
      return;
    }
    const s = data.status || {};
    const last = s.last_fetch && s.last_fetch !== "0001-01-01T00:00:00Z"
      ? new Date(s.last_fetch).toLocaleString("fr-FR")
      : "jamais";
    const errCls = s.last_error ? "bad" : "ok";
    const errTxt = s.last_error ? esc(s.last_error) : "aucune";
    el.innerHTML = `
      <div><span class="k">Statut:</span> <span class="ok">activé</span></div>
      <div><span class="k">Domaines bloqués:</span> <span class="v">${s.total_domains ?? 0}</span></div>
      <div><span class="k">Cycles de mise à jour:</span> <span class="v">${s.cycle_count ?? 0}</span></div>
      <div><span class="k">Dernier fetch:</span> <span class="v">${last}</span></div>
      <div><span class="k">Prochain dans:</span> <span class="v">${esc(s.update_every || "6h")}</span></div>
      <div><span class="k">Dernier erreur:</span> <span class="${errCls}">${errTxt}</span></div>`;
  } catch (err) {
    el.innerHTML = `<span class="bad">Erreur: ${esc(err.message)}</span>`;
  }
}

// ---- CVE : vérification de vulnérabilités connues ------------------------
document.getElementById("cve-form").addEventListener("submit", async (e) => {
  e.preventDefault();
  const fd = new FormData(e.target);
  const name = fd.get("name").trim();
  const version = fd.get("version").trim();
  if (!name) return;
  const results = document.getElementById("cve-results");
  results.innerHTML = `<li class="empty">Recherche en cours…</li>`;
  try {
    const resp = await postJSON("/api/cve", {
      software: [{ name, version }],
    });
    const alerts = resp.alerts || [];
    if (alerts.length === 0) {
      results.innerHTML = `<li class="empty">✓ Aucune CVE connue pour « ${esc(name)} ${esc(version)} » dans la fenêtre récente NVD.</li>`;
      return;
    }
    results.innerHTML = alerts
      .map((a) => {
        const sev = a.Severity || "LOW";
        return `
      <li>
        <div class="cve-head">
          <span class="cve-id">${esc(a.CVE)}</span>
          <span class="cve-sev ${sev}">${sev}</span>
        </div>
        <div class="cve-soft">${esc(a.Software)}</div>
        <div class="cve-desc">${esc(a.Description || "")}</div>
        <div class="cve-link"><a href="${esc(a.URL)}" target="_blank" rel="noopener">Détails NVD →</a></div>
      </li>`;
      })
      .join("");
  } catch (err) {
    results.innerHTML = `<li class="empty">Erreur: ${esc(err.message)}</li>`;
  }
});

// ---- scan ClamAV ----------------------------------------------------------
document.getElementById("scan-form").addEventListener("submit", async (e) => {
  e.preventDefault();
  const fd = new FormData(e.target);
  const path = fd.get("path").trim();
  if (!path) return;
  const el = document.getElementById("scan-result");
  el.innerHTML = `<span class="k">Scan en cours…</span> <span class="warn">(peut prendre plusieurs secondes)</span>`;
  try {
    const resp = await fetch(`${API}/api/scan?path=${encodeURIComponent(path)}`);
    if (!resp.ok) {
      const msg = await resp.text();
      el.innerHTML = `<span class="bad">Échec: ${esc(msg)}</span>`;
      return;
    }
    const r = await resp.json();
    const cls = r.Infected ? "bad" : "ok";
    const icon = r.Infected ? "⚠️" : "✓";
    let html = `<div><span class="${cls}">${icon} ${esc(r.Detail || "Résultat")}</span></div>`;
    if (r.Signature) {
      html += `<div><span class="k">Signature:</span> <span class="v">${esc(r.Signature)}</span></div>`;
    }
    html += `<div><span class="k">Fichier:</span> <span class="v">${esc(r.Path)}</span></div>`;
    el.innerHTML = html;
  } catch (err) {
    el.innerHTML = `<span class="bad">Erreur: ${esc(err.message)}</span>`;
  }
});

// ---- centre de sécurité : vue unifiée des protections -------------------
async function refreshSecurityCenter() {
  const summaryEl = document.getElementById("sec-summary");
  const gridEl = document.getElementById("sec-protections");
  try {
    const data = await getJSON("/api/security-center");
    const s = data.summary || {};
    const score = s.score || 0;
    const cls = score >= 75 ? "good" : score >= 40 ? "mid" : "bad";
    summaryEl.innerHTML = `
      <div class="big ${cls}">${score}%</div>
      <div class="label">Score de protection</div>
      <div class="details">
        <div class="stat active"><span class="n">${s.active || 0}</span> actives</div>
        <div class="stat limited"><span class="n">${s.limited || 0}</span> limitées</div>
        <div class="stat inactive"><span class="n">${s.inactive || 0}</span> inactives</div>
        <div class="stat"><span class="n">${s.error || 0}</span> en erreur</div>
      </div>`;

    const prots = data.protections || [];
    if (prots.length === 0) {
      gridEl.innerHTML = `<p class="empty">Aucune protection déclarée.</p>`;
      return;
    }
    const statusLabel = { active: "Actif", inactive: "Inactif", limited: "Limité", error: "Erreur" };
    gridEl.innerHTML = prots
      .map((p) => {
        let statsHtml = "";
        if (p.stats) {
          const entries = Object.entries(p.stats)
            .map(([k, v]) => `${esc(k)}: <strong>${v}</strong>`)
            .join(" · ");
          statsHtml = `<div class="stats">${entries}</div>`;
        }
        return `
      <div class="sec-card ${p.status}">
        <div class="head">
          <span class="icon">${esc(p.icon || "🔒")}</span>
          <span class="name">${esc(p.name)}</span>
          <span class="pill ${p.status}">${statusLabel[p.status] || p.status}</span>
        </div>
        <div class="cat">${esc(p.category || "")}</div>
        <div class="desc">${esc(p.description || "")}</div>
        ${statsHtml}
      </div>`;
      })
      .join("");
  } catch (err) {
    summaryEl.innerHTML = `<span class="bad">Erreur: ${esc(err.message)}</span>`;
  }
}

// ---- paramètres : mode simple + avancé ----------------------------------
const MODULES = [
  { key: "firewall",       label: "Pare-feu par application",   icon: "🛡️", simple: true },
  { key: "dns_sinkhole",   label: "DNS sinkhole (anti-pubs/trackers/malwares)", icon: "🌐", simple: true },
  { key: "anti_ads",       label: "Anti-publicités",            icon: "🚫", simple: false },
  { key: "anti_trackers",  label: "Anti-trackers",              icon: "🔍", simple: false },
  { key: "anti_malware",   label: "Anti-malware (DNS)",         icon: "🦠", simple: true },
  { key: "anti_phishing",  label: "Anti-phishing",              icon: "🎣", simple: false },
  { key: "anti_ransomware",label: "Anti-ransomware",            icon: "🔐", simple: false },
  { key: "av_scanner",     label: "Scanner ClamAV",             icon: "🔬", simple: false },
  { key: "yara_scanner",   label: "Scanner YARA",               icon: "🧬", simple: false },
  { key: "cve_feed",       label: "Veille CVE (NVD)",           icon: "📢", simple: false },
  { key: "threat_intel",   label: "Threat intelligence",        icon: "🕵️", simple: false },
  { key: "gamification",   label: "Gamification",               icon: "🎮", simple: false },
  { key: "auto_update",    label: "Auto-update des listes",     icon: "🔄", simple: false },
  { key: "freshclam",      label: "MAJ signatures ClamAV",      icon: "📥", simple: false },
];

async function loadSettings() {
  const body = document.getElementById("settings-body");
  const msgEl = document.getElementById("settings-msg");
  try {
    const s = await getJSON("/api/settings");
    // Section mode simple (toujours visible).
    let html = `<h3 style="margin-top:0">Mode simple</h3>`;
    html += settingsToggle("ui_mode", "Mode interface", s.ui_mode === "advanced" ? "Avancé" : "Simple", s.ui_mode === "advanced", "advanced");
    html += settingsToggle("auto_protect", "Protection automatique (recommandé)", "Active les protections recommandées", s.auto_protect, "auto_protect");
    html += settingsToggle("auto_update", "Mises à jour automatiques", "Rafraîchit les listes DNS/CVE toutes les 6h", s.auto_update, "auto_update");
    html += settingsToggle("cve_feed", "Veille des vulnérabilités (CVE)", "Alerte si un logiciel installé a une faille connue", s.cve_feed, "cve_feed");
    html += settingsToggle("threat_intel", "Threat intelligence", "Croise les IOC publics (Abuse.ch, OTX)", s.threat_intel, "threat_intel");
    html += settingsToggle("gamification", "Gamification", "Points, badges, streak de jours protégés", s.gamification, "gamification");

    // Section mode avancé (tous les modules).
    html += `<h3>Mode avancé — tous les modules</h3>`;
    for (const m of MODULES) {
      if (["ui_mode", "auto_protect", "auto_update", "cve_feed", "threat_intel", "gamification"].includes(m.key)) continue;
      const checked = !!s[m.key];
      html += settingsToggle(m.key, `${m.icon} ${m.label}`, "", checked, m.key);
    }

    // Réseau + thème.
    html += `<h3>Réseau & interface</h3>`;
    html += settingsSelect("profile", "Profil réseau", [
      { v: "home", t: "🏠 Maison" },
      { v: "office", t: "🏢 Bureau" },
      { v: "public", t: "🌍 Public (strict)" },
    ], s.profile);
    html += settingsSelect("theme", "Thème", [
      { v: "dark", t: "🌙 Sombre" },
      { v: "light", t: "☀️ Clair" },
      { v: "auto", t: "🖥️ Auto" },
    ], s.theme);
    html += settingsSelect("notifications", "Niveau de notifications", [
      { v: "minimal", t: "Critiques seulement" },
      { v: "normal", t: "Normal" },
      { v: "verbose", t: "Verbeux (tout)" },
    ], s.notifications);

    // Loopback : affiché verrouillé (non négociable).
    html += `<div class="set-row"><div><div class="set-label">🔒 Loopback uniquement</div><div class="set-hint">Le canal de contrôle n'écoute que sur 127.0.0.1 — non négociable</div></div><span class="set-locked">Verrouillé (activé)</span></div>`;

    body.innerHTML = html;
    msgEl.innerHTML = "";

    // Branche les change handlers.
    body.querySelectorAll("[data-setting]").forEach((el) => {
      el.addEventListener("change", () => saveSetting(el));
    });
  } catch (err) {
    body.innerHTML = "";
    msgEl.innerHTML = `<span class="bad">Erreur: ${esc(err.message)}</span>`;
  }
}

function settingsToggle(key, label, hint, checked, id) {
  return `<div class="set-row">
    <div>
      <div class="set-label">${esc(label)}</div>
      ${hint ? `<div class="set-hint">${esc(hint)}</div>` : ""}
    </div>
    <label class="switch">
      <input type="checkbox" data-setting="${esc(key)}" id="set-${esc(id)}" ${checked ? "checked" : ""} />
      <span class="slider"></span>
    </label>
  </div>`;
}

function settingsSelect(key, label, options, current) {
  const opts = options.map((o) => `<option value="${esc(o.v)}" ${o.v === current ? "selected" : ""}>${esc(o.t)}</option>`).join("");
  return `<div class="set-row">
    <div><div class="set-label">${esc(label)}</div></div>
    <select data-setting="${esc(key)}" class="set-select">${opts}</select>
  </div>`;
}

async function saveSetting(el) {
  const key = el.dataset.setting;
  let value;
  if (el.type === "checkbox") value = el.checked;
  else value = el.value;
  const msgEl = document.getElementById("settings-msg");
  try {
    await postJSON("/api/settings", { [key]: value });
    msgEl.innerHTML = `<span class="ok">✓ Enregistré</span>`;
    setTimeout(() => { msgEl.innerHTML = ""; }, 2000);
    // Si le thème change, on l'applique immédiatement.
    if (key === "theme") { localStorage.setItem("faillefox-theme", value); applyTheme(value); }
  } catch (err) {
    msgEl.innerHTML = `<span class="bad">Erreur: ${esc(err.message)}</span>`;
  }
}

// ---- bootstrap ------------------------------------------------------------
(async function init() {
  try {
    refreshSecurityCenter();
    await loadApps();
    await refreshRules();
    await refreshStatus();
    refreshUpdater();
    loadSettings();
    startEventStream();
    // Rafraîchit le centre de sécurité et l'auto-update toutes les 30s.
    setInterval(refreshSecurityCenter, 30000);
    setInterval(refreshUpdater, 30000);
  } catch (err) {
    document.getElementById("status-text").textContent = "Erreur: " + err.message;
    document.getElementById("status-dot").className = "dot bad";
  }
})();
