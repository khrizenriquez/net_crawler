export const formatBytes = (bytes = 0) =>
  bytes > 1_000_000_000
    ? `${(bytes / 1_000_000_000).toFixed(2)} GB`
    : bytes > 1_000_000
      ? `${(bytes / 1_000_000).toFixed(1)} MB`
      : `${(bytes / 1_000).toFixed(1)} KB`;

export const coverage = (value: string) =>
  ({
    wan_total: "WAN TOTAL",
    wifi_observed: "WI-FI OBSERVADO",
    partial: "PARCIAL",
    not_decryptable: "NO DESCIFRABLE",
    out_of_scope: "FUERA DE ALCANCE",
  })[value] ?? value.toUpperCase();

export const when = (value: string) =>
  new Intl.DateTimeFormat("es-GT", { dateStyle: "short", timeStyle: "short" }).format(new Date(value));
