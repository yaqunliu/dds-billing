import { useState, useEffect } from "react";
import { useUrlParams } from "../hooks/useUrlParams";
import { getConfig, createOrder, type AppConfig } from "../api";
import QRCodeModal from "../components/QRCode";
import CardCheckout from "../components/CardCheckout";
import {
  QUICK_AMOUNTS,
  CARD_QUICK_AMOUNTS,
  PAYMENT_TYPE_CONFIG,
} from "../utils/constant";
import { normalizeLang } from "../utils/i18n";
import { PAY_MESSAGES, pickLocale } from "../utils/locale";

export default function Pay() {
  const { token, theme, lang } = useUrlParams();
  const [config, setConfig] = useState<AppConfig | null>(null);
  const [amount, setAmount] = useState<number | string>("");
  const [paymentType, setPaymentType] = useState<"wxpay" | "alipay" | "card">(
    "alipay",
  );
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  // QR code modal state
  const [orderNo, setOrderNo] = useState("");
  const [qrCodeUrl, setQrCodeUrl] = useState("");
  const [payUrl, setPayUrl] = useState("");
  const [expiresAt, setExpiresAt] = useState("");
  const [showQR, setShowQR] = useState(false);

  // 信用卡内嵌收银台
  const [clientSecret, setClientSecret] = useState("");
  const [showCard, setShowCard] = useState(false);
  const [chargeAmount, setChargeAmount] = useState(0);
  const [chargeCurrency, setChargeCurrency] = useState("usd");

  useEffect(() => {
    getConfig()
      .then((res) => {
        if (res.data.code === 0) setConfig(res.data.data);
      })
      .catch(() => {});
  }, []);

  const isDark = theme === "dark";
  const appLang = normalizeLang(lang);
  const t = pickLocale(PAY_MESSAGES, appLang);
  const enabledTypes = config?.enabled_types ?? ["wxpay", "alipay"];

  const baseMin = config?.min_amount ?? 10;
  const cardMin = config?.card_min_amount ?? 200;
  const maxAmount = config?.max_amount ?? 20000;

  // 信用卡使用独立的最低起充金额
  const isCard = paymentType === "card";
  const effectiveMin = isCard ? cardMin : baseMin;
  const quickAmounts = isCard ? CARD_QUICK_AMOUNTS : QUICK_AMOUNTS;

  const isAmountValid =
    amount !== "" &&
    !isNaN(Number(amount)) &&
    Number(amount) >= effectiveMin &&
    Number(amount) <= maxAmount;

  const policyItems = t.policyItems(baseMin);

  const handleAmountChange = (val: number | string) => {
    setAmount(val);
    setError("");
  };

  // 切换支付方式：若切到信用卡且当前金额低于信用卡下限，清空金额避免无效提交
  const handlePaymentTypeChange = (type: "wxpay" | "alipay" | "card") => {
    setPaymentType(type);
    setError("");
    if (type === "card" && amount !== "" && Number(amount) < cardMin) {
      setAmount("");
    }
  };

  const handleSubmit = async () => {
    if (!amount || !isAmountValid) {
      setError(t.amountRange(effectiveMin, maxAmount));
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
        lang: appLang,
      });
      if (res.data.code === 0) {
        // 信用卡：在页面内（iframe 内）打开内嵌收银台，不跳出顶层
        if (paymentType === "card") {
          if (!config?.publishable_key || !res.data.data.client_secret) {
            setError(t.createOrderFailed);
            return;
          }
          setOrderNo(res.data.data.order_no);
          setClientSecret(res.data.data.client_secret);
          setChargeAmount(res.data.data.charge_amount ?? Number(amount));
          setChargeCurrency(res.data.data.charge_currency ?? "cny");
          setShowCard(true);
          return;
        }
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
      className={`min-h-screen transition-colors ${
        isDark
          ? "bg-slate-950 text-gray-100"
          : "bg-[radial-gradient(circle_at_top,_rgba(236,253,245,1),_rgba(248,250,252,1)_45%,_rgba(240,253,250,1)_100%)] text-slate-900"
      }`}
    >
      <div className="mx-auto max-w-6xl px-4 py-6 sm:px-6 sm:py-8">
        <div
          className={`mb-6 rounded-[28px] border p-6 shadow-[0_18px_50px_rgba(16,185,129,0.12)] backdrop-blur md:p-8 ${
            isDark
              ? "border-slate-800 bg-slate-900/90 shadow-none"
              : "border-emerald-100/80 bg-white/85"
          }`}
        >
          <div className="mb-5 flex flex-wrap items-center justify-between gap-3">
            <div>
              <p
                className={`mb-2 inline-flex rounded-full px-3 py-1 text-xs font-semibold ${
                  isDark
                    ? "bg-emerald-500/15 text-emerald-300"
                    : "bg-emerald-50 text-emerald-700"
                }`}
              >
                {t.policyTitle}
              </p>
              <h1 className="text-xl font-semibold sm:text-2xl">
                {t.policySubtitle}
              </h1>
            </div>
          </div>

          <div
            className={`space-y-3 text-sm leading-7 sm:text-base ${
              isDark ? "text-slate-300" : "text-slate-700"
            }`}
          >
            {policyItems.map((item) => (
              <p key={item}>{item}</p>
            ))}
          </div>
        </div>

        <div
          className={`rounded-[32px] border p-5 shadow-[0_22px_60px_rgba(16,185,129,0.14)] md:p-8 ${
            isDark
              ? "border-slate-800 bg-slate-900/95 shadow-none"
              : "border-emerald-100/80 bg-white/90"
          }`}
        >
          <div className="mb-6 flex items-start justify-between gap-4">
            <div>
              <p
                className={`mb-2 inline-flex rounded-full px-3 py-1 text-xs font-semibold ${
                  isDark
                    ? "bg-sky-500/15 text-sky-300"
                    : "bg-sky-50 text-sky-700"
                }`}
              >
                {t.title}
              </p>
              <h2 className="text-2xl font-semibold sm:text-3xl">
                {amount
                  ? `¥${Number(amount).toFixed(2)}`
                  : `¥${effectiveMin} 起充`}
              </h2>
              <p
                className={`mt-2 text-sm ${
                  isDark ? "text-slate-400" : "text-slate-500"
                }`}
              >
                {appLang === "zh"
                  ? `最低充值 ¥${effectiveMin}，单笔范围 ¥${effectiveMin} - ¥${maxAmount}`
                  : `Minimum recharge ¥${effectiveMin}, range ¥${effectiveMin} - ¥${maxAmount}`}
              </p>
            </div>
          </div>

          <div
            className={`rounded-[28px] border p-5 sm:p-6 ${
              isDark
                ? "border-slate-800 bg-slate-950/70"
                : "border-emerald-100 bg-[linear-gradient(180deg,rgba(240,253,250,0.95),rgba(255,255,255,0.96))]"
            }`}
          >
            <div className="mb-6">
              <label
                className={`mb-3 block text-sm font-medium ${
                  isDark ? "text-slate-300" : "text-slate-700"
                }`}
              >
                {t.paymentType}
              </label>
              <div className="grid grid-cols-3 gap-2 sm:gap-3">
                {(enabledTypes as Array<keyof typeof PAYMENT_TYPE_CONFIG>).map(
                  (type) => {
                    const option = PAYMENT_TYPE_CONFIG[type];
                    const label = option.getLabel(t);

                    return (
                      <button
                        key={type}
                        onClick={() => handlePaymentTypeChange(type)}
                        className={`flex flex-col items-center justify-center gap-1.5 rounded-2xl border px-2 py-3 text-sm font-medium transition-all ${
                          paymentType === type
                            ? isDark
                              ? option.activeClass
                              : `${option.activeLightClass} bg-white`
                            : isDark
                              ? `bg-slate-900 text-slate-200 border-slate-700 ${option.inactiveHoverClass}`
                              : `bg-white text-slate-700 border-emerald-100 shadow-[0_8px_24px_rgba(15,23,42,0.06)] ${option.inactiveLightHoverClass}`
                        }`}
                      >
                        <img
                          src={option.icon}
                          alt={label}
                          className="h-6 w-6 rounded"
                        />
                        {label}
                      </button>
                    );
                  },
                )}
              </div>
              {isCard && (
                <p
                  className={`mt-2 text-xs ${
                    isDark ? "text-indigo-300" : "text-indigo-600"
                  }`}
                >
                  {t.cardMinTip(cardMin)}
                </p>
              )}
            </div>

            <div className="mb-6">
              <label
                className={`mb-3 block text-sm font-medium ${
                  isDark ? "text-slate-300" : "text-slate-700"
                }`}
              >
                {t.amount}
              </label>
              <div className="grid grid-cols-2 gap-3 sm:grid-cols-3">
                {quickAmounts.map((val) => (
                  <button
                    key={val}
                    onClick={() => handleAmountChange(val)}
                    className={`rounded-2xl border px-4 py-3 text-base font-semibold transition-all sm:text-lg ${
                      amount === val
                        ? isDark
                          ? "border-sky-500 bg-sky-500/15 text-sky-200 shadow-[0_10px_30px_rgba(14,165,233,0.16)]"
                          : "border-sky-500 bg-white text-sky-700 shadow-[0_12px_30px_rgba(14,165,233,0.14)]"
                        : isDark
                          ? "border-slate-700 bg-slate-900 text-slate-200 hover:border-sky-500/70"
                          : "border-white bg-white text-slate-700 shadow-[0_8px_24px_rgba(15,23,42,0.06)] hover:border-emerald-200 hover:bg-emerald-50/50"
                    }`}
                  >
                    ¥{val}
                  </button>
                ))}
              </div>
            </div>

            <div className="mb-6">
              <label
                className={`mb-2 block text-sm font-medium ${
                  isDark ? "text-slate-300" : "text-slate-700"
                }`}
              >
                {t.customAmount}
              </label>
              <div className="relative">
                <span
                  className={`absolute left-4 top-1/2 -translate-y-1/2 text-lg ${
                    isDark ? "text-slate-400" : "text-slate-400"
                  }`}
                >
                  ¥
                </span>
                <input
                  type="number"
                  value={amount}
                  onChange={(e) => handleAmountChange(e.target.value)}
                  placeholder={`${effectiveMin} - ${maxAmount}`}
                  className={`w-full rounded-2xl border py-3 pr-4 pl-10 text-lg transition-colors focus:outline-none focus:ring-4 ${
                    isDark
                      ? "border-slate-700 bg-slate-900 text-slate-100 placeholder-slate-500 focus:border-sky-500 focus:ring-sky-500/10"
                      : "border-emerald-100 bg-white text-slate-900 placeholder-slate-400 focus:border-sky-400 focus:ring-sky-500/10"
                  }`}
                />
              </div>
            </div>

            {error && (
              <div
                className="mb-4 rounded-2xl border border-red-500/20 bg-red-500/10 p-3 text-sm text-red-500"
                style={{ wordWrap: "break-word" }}
              >
                {error}
              </div>
            )}

            <button
              onClick={handleSubmit}
              disabled={loading || !isAmountValid}
              className={`w-full rounded-full py-4 text-lg font-bold transition-all ${
                loading || !isAmountValid
                  ? "cursor-not-allowed bg-slate-300 text-slate-100"
                  : isDark
                    ? "bg-sky-500 text-white shadow-[0_16px_40px_rgba(14,165,233,0.24)] hover:bg-sky-400"
                    : "bg-white text-slate-900 shadow-[0_16px_40px_rgba(16,185,129,0.18)] ring-1 ring-emerald-100 hover:bg-emerald-50"
              }`}
            >
              {loading
                ? t.creatingOrder
                : amount
                  ? t.payAmount(Number(amount))
                  : t.selectAmount}
            </button>
          </div>
        </div>
      </div>

      {showQR && paymentType !== "card" && (
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

      {showCard && config?.publishable_key && clientSecret && (
        <CardCheckout
          publishableKey={config.publishable_key}
          clientSecret={clientSecret}
          orderNo={orderNo}
          chargeAmount={chargeAmount}
          chargeCurrency={chargeCurrency}
          cnyAmount={Number(amount)}
          isDark={isDark}
          lang={appLang}
          onClose={() => setShowCard(false)}
        />
      )}
    </div>
  );
}
