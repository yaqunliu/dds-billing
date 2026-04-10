import { useEffect, useState } from "react";
import { useUrlParams } from "../hooks/useUrlParams";
import { getOrders, type OrderData } from "../api";
import {
  formatDateTime,
  getOrderStatusLabel,
  normalizeLang,
} from "../utils/i18n";
import { ORDERS_MESSAGES, pickLocale } from "../utils/locale";

export default function Orders() {
  const { token, user_id, theme, lang } = useUrlParams();
  const isDark = theme === "dark";
  const appLang = normalizeLang(lang);
  const t = pickLocale(ORDERS_MESSAGES, appLang);

  const [orders, setOrders] = useState<OrderData[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [loading, setLoading] = useState(true);

  const pageSize = 20;

  useEffect(() => {
    if (!token) {
      setLoading(false);
      return;
    }
    setLoading(true);
    getOrders(token, page, pageSize)
      .then((res) => {
        if (res.data.code === 0) {
          setOrders(res.data.data.list || []);
          setTotal(res.data.data.total);
        }
      })
      .catch(() => {})
      .finally(() => setLoading(false));
  }, [token, user_id, page]);

  const totalPages = Math.ceil(total / pageSize);

  const statusColorMap: Record<string, string> = {
    pending: "bg-yellow-100 text-yellow-700",
    paid: "bg-blue-100 text-blue-700",
    recharging: "bg-blue-100 text-blue-700",
    completed: "bg-green-100 text-green-700",
    failed: "bg-red-100 text-red-700",
    expired: "bg-gray-100 text-gray-600",
  };

  return (
    <div
      className={`min-h-screen transition-colors ${isDark ? "bg-gray-900 text-gray-100" : "bg-gray-50 text-gray-900"}`}
    >
      <div className="max-w-lg mx-auto px-4 py-8">
        <h1 className="text-2xl font-bold mb-6">{t.title}</h1>

        {loading ? (
          <p
            className={`text-center py-8 ${isDark ? "text-gray-400" : "text-gray-500"}`}
          >
            {t.loading}
          </p>
        ) : orders.length === 0 ? (
          <p
            className={`text-center py-8 ${isDark ? "text-gray-400" : "text-gray-500"}`}
          >
            {t.empty}
          </p>
        ) : (
          <div className="space-y-3">
            {orders.map((order) => {
              const label = getOrderStatusLabel(order.status, appLang);
              const color =
                statusColorMap[order.status] ?? "bg-gray-100 text-gray-600";
              return (
                <div
                  key={order.order_no}
                  className={`rounded-xl p-4 ${isDark ? "bg-gray-800" : "bg-white shadow-sm"}`}
                >
                  <div className="flex items-center justify-between mb-2">
                    <span className="font-bold text-lg">
                      ¥{order.amount.toFixed(2)}
                    </span>
                    <span
                      className={`px-2 py-0.5 rounded-full text-xs font-medium ${color}`}
                    >
                      {label}
                    </span>
                  </div>
                  <div
                    className={`flex items-center justify-between text-xs ${isDark ? "text-gray-400" : "text-gray-500"}`}
                  >
                    <span className="font-mono">{order.order_no}</span>
                    <span>
                      {formatDateTime(
                        order.paid_at || order.expires_at,
                        appLang,
                      )}
                    </span>
                  </div>
                </div>
              );
            })}
          </div>
        )}

        {/* Pagination*/}
        {totalPages > 1 && (
          <div className="flex items-center justify-center gap-4 mt-6">
            <button
              onClick={() => setPage((p) => Math.max(1, p - 1))}
              disabled={page <= 1}
              className={`px-4 py-2 rounded-lg text-sm transition-colors
                ${
                  page <= 1
                    ? "opacity-40 cursor-not-allowed"
                    : isDark
                      ? "bg-gray-800 hover:bg-gray-700"
                      : "bg-white hover:bg-gray-100 shadow-sm"
                }`}
            >
              {t.prev}
            </button>
            <span
              className={`text-sm ${isDark ? "text-gray-400" : "text-gray-500"}`}
            >
              {page} / {totalPages}
            </span>
            <button
              onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
              disabled={page >= totalPages}
              className={`px-4 py-2 rounded-lg text-sm transition-colors
                ${
                  page >= totalPages
                    ? "opacity-40 cursor-not-allowed"
                    : isDark
                      ? "bg-gray-800 hover:bg-gray-700"
                      : "bg-white hover:bg-gray-100 shadow-sm"
                }`}
            >
              {t.next}
            </button>
          </div>
        )}
      </div>
    </div>
  );
}
