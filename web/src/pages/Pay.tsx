import { useState, useEffect } from "react";
import { useUrlParams } from "../hooks/useUrlParams";
import { getConfig, createOrder, type AppConfig } from "../api";
import QRCodeModal from "../components/QRCode";
import { QUICK_AMOUNTS, PAYMENT_TYPE_CONFIG } from "../utils/constant";
import { normalizeLang } from "../utils/i18n";
import { PAY_MESSAGES, pickLocale } from "../utils/locale";

export default function Pay() {
  const { token, theme, lang } = useUrlParams();
  const [config, setConfig] = useState<AppConfig | null>(null);
  const [amount, setAmount] = useState<number | string>("");
  const [paymentType, setPaymentType] = useState<"wxpay" | "alipay">("wxpay");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  // QR code modal state
  const [orderNo, setOrderNo] = useState("");
  const [qrCodeUrl, setQrCodeUrl] = useState("");
  const [payUrl, setPayUrl] = useState("");
  const [expiresAt, setExpiresAt] = useState("");
  const [showQR, setShowQR] = useState(false);

  useEffect(() => {
    getConfig()
      .then((res) => {
        if (res.data.code === 0) setConfig(res.data.data);
      })
      .catch(() => {});
  }, []);

  const isAmountValid =
    amount !== "" &&
    !isNaN(Number(amount)) &&
    Number(amount) >= (config?.min_amount ?? 1) &&
    Number(amount) <= (config?.max_amount ?? 20000);

  const enabledTypes = config?.enabled_types ?? ["wxpay", "alipay"];
  const isDark = theme === "dark";
  const appLang = normalizeLang(lang);
  const t = pickLocale(PAY_MESSAGES, appLang);

  const handleAmountChange = (val: number | string) => {
    setAmount(val);
    setError("");
  };

  const handleSubmit = async () => {
    if (!amount || !isAmountValid) {
      setError(
        t.amountRange(config?.min_amount ?? 1, config?.max_amount ?? 20000),
      );
      return;
    }
    if (!token) {
      setError(t.missingToken);
      return;
    }

    setLoading(true);
    setError("");
    try {
      const res = await createOrder({
        token,
        amount: Number(amount),
        payment_type: paymentType,
      });
      if (res.data.code === 0) {
        setOrderNo(res.data.data.order_no);
        setQrCodeUrl(res.data.data.qr_code_url);
        setPayUrl(res.data.data.pay_url);
        setExpiresAt(res.data.data.expires_at);
        setShowQR(true);
      } else {
        setError(res.data.message || t.createOrderFailed);
      }
    } catch (err: any) {
      setError(err.response?.data?.message || t.networkError);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div
      className={`min-h-screen transition-colors ${isDark ? "bg-gray-900 text-gray-100" : "bg-gray-50 text-gray-900"}`}
    >
      <div className="max-w-md mx-auto px-4 py-8">
        <h1 className="text-2xl font-bold text-center mb-8">{t.title}</h1>

        <div className="mb-6">
          <label
            className={`block text-sm font-medium mb-3 ${isDark ? "text-gray-300" : "text-gray-700"}`}
          >
            {t.amount}
          </label>
          <div className="grid grid-cols-3 gap-3">
            {QUICK_AMOUNTS.map((val) => (
              <button
                key={val}
                onClick={() => handleAmountChange(val)}
                className={`py-3 rounded-lg text-lg font-medium transition-all border
                  ${
                    amount === val
                      ? "bg-blue-600 text-white border-blue-600 shadow-md"
                      : isDark
                        ? "bg-gray-800 text-gray-200 border-gray-700 hover:border-blue-500"
                        : "bg-white text-gray-700 border-gray-200 hover:border-blue-400"
                  }`}
              >
                ¥{val}
              </button>
            ))}
          </div>
        </div>

        <div className="mb-6">
          <label
            className={`block text-sm font-medium mb-2 ${isDark ? "text-gray-300" : "text-gray-700"}`}
          >
            {t.customAmount}
          </label>
          <div className="relative">
            <span
              className={`absolute left-3 top-1/2 -translate-y-1/2 text-lg ${isDark ? "text-gray-400" : "text-gray-500"}`}
            >
              ¥
            </span>
            <input
              type="number"
              value={amount}
              onChange={(e) => handleAmountChange(e.target.value)}
              placeholder={`${config?.min_amount ?? 1} - ${config?.max_amount ?? 20000}`}
              className={`w-full pl-8 pr-4 py-3 rounded-lg border text-lg transition-colors
                ${
                  isDark
                    ? "bg-gray-800 border-gray-700 text-gray-100 placeholder-gray-500 focus:border-blue-500"
                    : "bg-white border-gray-200 text-gray-900 placeholder-gray-400 focus:border-blue-500"
                } focus:outline-none focus:ring-2 focus:ring-blue-500/20`}
            />
          </div>
        </div>

        <div className="mb-8">
          <label
            className={`block text-sm font-medium mb-3 ${isDark ? "text-gray-300" : "text-gray-700"}`}
          >
            {t.paymentType}
          </label>
          <div className="flex gap-3">
            {(enabledTypes as Array<keyof typeof PAYMENT_TYPE_CONFIG>).map(
              (type) => {
                const option = PAYMENT_TYPE_CONFIG[type];
                const label = option.getLabel(t);

                return (
                  <button
                    key={type}
                    onClick={() => setPaymentType(type)}
                    className={`flex-1 py-3 rounded-lg font-medium transition-all border-2 flex items-center justify-center gap-2
                    ${
                      paymentType === type
                        ? isDark
                          ? option.activeClass
                          : option.activeLightClass
                        : isDark
                          ? `bg-gray-800 text-gray-200 border-gray-700 ${option.inactiveHoverClass}`
                          : `bg-white text-gray-700 border-gray-200 ${option.inactiveLightHoverClass}`
                    }`}
                  >
                    <img
                      src={option.icon}
                      alt={label}
                      className="w-6 h-6 rounded"
                    />
                    {label}
                  </button>
                );
              },
            )}
          </div>
        </div>

        {error && (
          <div
            className="mb-4 p-3 rounded-lg bg-red-500/10 border border-red-500/20 text-red-500 text-sm"
            style={{ wordWrap: "break-word" }}
          >
            {error}
          </div>
        )}

        <button
          onClick={handleSubmit}
          disabled={loading || !isAmountValid}
          className={`w-full py-4 rounded-lg text-lg font-bold transition-all
            ${
              loading || !isAmountValid
                ? "bg-gray-400 text-gray-200 cursor-not-allowed"
                : "bg-blue-600 text-white hover:bg-blue-700 active:bg-blue-800 shadow-lg"
            }`}
        >
          {loading
            ? t.creatingOrder
            : amount
              ? t.payAmount(Number(amount))
              : t.selectAmount}
        </button>
      </div>

      {showQR && (
        <QRCodeModal
          orderNo={orderNo}
          qrCodeUrl={qrCodeUrl}
          payUrl={payUrl}
          expiresAt={expiresAt}
          paymentType={paymentType}
          isDark={isDark}
          lang={appLang}
          onClose={() => setShowQR(false)}
        />
      )}
    </div>
  );
}
