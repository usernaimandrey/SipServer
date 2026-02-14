import React from "react";
import { pretty } from "../api";

type Props = {
  rows: any[];
  preferCols?: string[]; // необязательно: важные колонки первыми
};

function buildCols(rows: any[], preferCols?: string[]) {
  const all = Array.from(new Set(rows.flatMap((r) => Object.keys(r || {}))));
  if (!preferCols?.length) return all;
  const prefer = preferCols.filter((c) => all.includes(c));
  const rest = all.filter((c) => !prefer.includes(c));
  return [...prefer, ...rest];
}

export default function ResponsiveTable({ rows, preferCols }: Props) {
  const cols = buildCols(rows, preferCols);

  // Рендер значения: объекты/длинные строки — в <pre> с переносами
  function renderValue(v: any) {
    const isObj = v && typeof v === "object";
    const text = isObj ? JSON.stringify(v, null, 2) : pretty(v);

    if (isObj || text.length > 80) {
      return <pre className="cellPre" style={{ margin: 0 }}>{text}</pre>;
    }
    return <span className="cell" title={text}>{text}</span>;
  }

  return (
    <div>
      {/* Desktop/tablet */}
      <div className="tableWrap">
        <table className="table">
          <thead>
            <tr>{cols.map((c) => <th key={c}>{c}</th>)}</tr>
          </thead>
          <tbody>
            {rows.map((r, i) => (
              <tr key={i}>
                {cols.map((c) => <td key={c}>{renderValue(r?.[c])}</td>)}
              </tr>
            ))}
            {!rows.length && (
              <tr><td colSpan={cols.length}><small className="muted">No data</small></td></tr>
            )}
          </tbody>
        </table>
      </div>

      {/* Mobile/cards */}
      <div className="cards" style={{ display: "none" }}>
        {rows.map((r, i) => (
          <div className="cardRow" key={i}>
            <div className="kv">
              {cols.map((c) => (
                <React.Fragment key={c}>
                  <div className="k">{c}</div>
                  <div className="v">{renderValue(r?.[c])}</div>
                </React.Fragment>
              ))}
            </div>
          </div>
        ))}
        {!rows.length && <small className="muted">No data</small>}
      </div>
    </div>
  );
}
