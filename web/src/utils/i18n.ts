export type AppLang = "zh" | "en";
import { STATUS_LABELS } from "./locale";

export function normalizeLang(lang?: string): AppLang {
  return lang?.toLowerCase().startsWith("en") ? "en" : "zh";
}

export function getOrderStatusLabel(status: string, lang: string): string {
  const appLang = normalizeLang(lang);
  return STATUS_LABELS[appLang][status] ?? status;
}

export function formatDateTime(value: string | number | Date, lang: string): string {
  const locale = normalizeLang(lang) === "en" ? "en-US" : "zh-CN";
  return new Date(value).toLocaleString(locale);
}
