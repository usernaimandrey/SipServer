import React, { useEffect, useState } from "react";
import { apiFetch } from "../api";
import ResponsiveTable from "../components/ResponsiveTable";

export default function CallJournals() {
  const [rows, setRows] = useState<any[]>([]);
  const [err, setErr] = useState("");
  const [busy, setBusy] = useState(false);

  async function load() {
    setErr("");
    setBusy(true);
    try {
      setRows(await apiFetch<any[]>("/api/call_journals"));
    } catch (e: any) {
      setErr(e.message || "load error");
    } finally {
      setBusy(false);
    }
  }

  useEffect(() => { load(); }, []);

  return (
    <div>
      <div className="row">
        <button onClick={load} disabled={busy}>Reload</button>
      </div>

      {err && <pre className="error">{err}</pre>}

      <div style={{ marginTop: 14 }}>
        <ResponsiveTable
          rows={rows}
          preferCols={["id", "call_id", "from", "to", "direction", "status", "started_at", "ended_at", "duration_sec"]}
        />
      </div>
    </div>
  );
}
