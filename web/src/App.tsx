import { useMemo, useState } from "react";
import Users from "./pages/Users";
import Sessions from "./pages/Sessions";
import CallJournals from "./pages/CallJournals";

type Tab = "users" | "sessions" | "journals";

export default function App() {
  const [tab, setTab] = useState<Tab>("users");
  const title = useMemo(() => {
    if (tab === "users") return "Users";
    if (tab === "sessions") return "Sessions";
    return "Call journals";
  }, [tab]);

  return (
    <div className="container">
      <h1 style={{ marginTop: 8, marginBottom: 10 }}>SipServer Admin</h1>
      <div className="tabs">
        <button className={`tab ${tab === "users" ? "active" : ""}`} onClick={() => setTab("users")}>Users</button>
        <button className={`tab ${tab === "sessions" ? "active" : ""}`} onClick={() => setTab("sessions")}>Sessions</button>
        <button className={`tab ${tab === "journals" ? "active" : ""}`} onClick={() => setTab("journals")}>Call journals</button>
      </div>

      <div className="card">
        <h2 style={{ marginTop: 0 }}>{title}</h2>
        {tab === "users" && <Users />}
        {tab === "sessions" && <Sessions />}
        {tab === "journals" && <CallJournals />}
        <div style={{ marginTop: 14 }}>
          <small className="muted">
            API: <code>/api/*</code>. В дев-режиме React на <code>:5173</code> проксирует в Go <code>:8080</code>.
          </small>
        </div>
      </div>
    </div>
  );
}
