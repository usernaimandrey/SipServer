import { useEffect, useState } from "react";
import { apiFetch } from "../api";

type User = {
  id: number;
  login: string;
  role: "admin" | "user";
  config: { call_schema: "redirect" | "proxy" };
};

export default function Users() {
  const [items, setItems] = useState<User[]>([]);
  const [err, setErr] = useState("");
  const [busy, setBusy] = useState(false);

  const [form, setForm] = useState({
    login: "",
    role: "user" as "user" | "admin",
    call_schema: "redirect" as "redirect" | "proxy",
  });

  async function load() {
    setErr("");
    setBusy(true);
    try {
      setItems(await apiFetch<User[]>("/api/users"));
    } catch (e: any) {
      setErr(e.message || "load error");
    } finally {
      setBusy(false);
    }
  }

  useEffect(() => { load(); }, []);

  async function create() {
    setErr("");
    setBusy(true);
    try {
      await apiFetch<User>("/api/users", {
        method: "POST",
        body: JSON.stringify({
          login: form.login.trim(),
          role: form.role,
          config: { call_schema: form.call_schema },
        }),
      });
      setForm({ ...form, login: "" });
      await load();
    } catch (e: any) {
      setErr(e.message || "create error");
    } finally {
      setBusy(false);
    }
  }

  async function edit(u: User) {
    const login = prompt("login:", u.login) ?? u.login;
    const role = (prompt("role (admin/user):", u.role) ?? u.role) as any;
    const schema = (prompt("call_schema (redirect/proxy):", u.config.call_schema) ?? u.config.call_schema) as any;

    setErr("");
    setBusy(true);
    try {
      await apiFetch<User>(`/api/users/${u.id}`, {
        method: "PUT",
        body: JSON.stringify({
          login,
          role,
          config: { call_schema: schema },
        }),
      });
      await load();
    } catch (e: any) {
      setErr(e.message || "update error");
    } finally {
      setBusy(false);
    }
  }

  return (
    <div>
      <div className="row">
        <div>
          <label>login</label>
          <input value={form.login} onChange={(e) => setForm({ ...form, login: e.target.value })} />
        </div>
        <div>
          <label>role</label>
          <select value={form.role} onChange={(e) => setForm({ ...form, role: e.target.value as any })}>
            <option value="user">user</option>
            <option value="admin">admin</option>
          </select>
        </div>
        <div>
          <label>call_schema</label>
          <select value={form.call_schema} onChange={(e) => setForm({ ...form, call_schema: e.target.value as any })}>
            <option value="redirect">redirect</option>
            <option value="proxy">proxy</option>
          </select>
        </div>
        <button onClick={create} disabled={busy || !form.login.trim()}>Create</button>
        <button onClick={load} disabled={busy}>Reload</button>
      </div>

      {err && <pre className="error">{err}</pre>}

      <table style={{ marginTop: 14 }}>
        <thead>
          <tr>
            <th>id</th>
            <th>login</th>
            <th>role</th>
            <th>call_schema</th>
            <th />
          </tr>
        </thead>
        <tbody>
          {items.map((u) => (
            <tr key={u.id}>
              <td>{u.id}</td>
              <td>{u.login}</td>
              <td>{u.role}</td>
              <td>{u.config?.call_schema}</td>
              <td><button onClick={() => edit(u)} disabled={busy}>Edit</button></td>
            </tr>
          ))}
          {!items.length && (
            <tr><td colSpan={5}><small className="muted">No users</small></td></tr>
          )}
        </tbody>
      </table>
    </div>
  );
}
