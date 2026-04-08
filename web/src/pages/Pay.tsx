import { useState, useEffect } from "react";
import { useUrlParams } from "../hooks/useUrlParams";
import { getConfig, createOrder, type AppConfig } from "../api";
import QRCodeModal from "../components/QRCode";
import alipayIcon from "../assets/alipay.jpg";
import wechatIcon from "../assets/wechat.png";

const QUICK_AMOUNTS = [10, 20, 50, 100, 200, 500];

export default function Pay() {
  const { token, theme } = useUrlParams();
  const [config, setConfig] = useState<AppConfig | null>(null);
  const [selectedAmount, setSelectedAmount] = useState<number | null>(null);
  const [customAmount, setCustomAmount] = useState("");
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

  const amount = customAmount ? parseFloat(customAmount) : selectedAmount;
  const isAmountValid =
    amount !== null &&
    !isNaN(amount!) &&
    amount! >= (config?.min_amount ?? 1) &&
    amount! <= (config?.max_amount ?? 20000);

  const enabledTypes = config?.enabled_types ?? ["wxpay", "alipay"];

  const handleSelectAmount = (val: number) => {
    setSelectedAmount(val);
    setCustomAmount("");
    setError("");
  };

  const handleCustomAmountChange = (val: string) => {
    setCustomAmount(val);
    setSelectedAmount(null);
    setError("");
  };

  const handleSubmit = async () => {
    if (!amount || !isAmountValid) {
      setError(
        `请输入 ${config?.min_amount ?? 1} - ${config?.max_amount ?? 20000} 之间的金额`,
      );
      return;
    }
    if (!token) {
      setError("缺少用户凭证，请从平台入口进入");
      return;
    }

    setLoading(true);
    setError("");
    try {
      const res = await createOrder({
        token,
        amount,
        payment_type: paymentType,
      });
      if (res.data.code === 0) {
        setOrderNo(res.data.data.order_no);
        setQrCodeUrl(res.data.data.qr_code_url);
        setPayUrl(res.data.data.pay_url);
        setExpiresAt(res.data.data.expires_at);
        setShowQR(true);
      } else {
        setError(res.data.message || "创建订单失败");
      }
    } catch (err: any) {
      setError(err.response?.data?.message || "网络错误，请重试");
    } finally {
      setLoading(false);
    }
  };

  const isDark = theme === "dark";

  return (
    <div
      className={`min-h-screen transition-colors ${isDark ? "bg-gray-900 text-gray-100" : "bg-gray-50 text-gray-900"}`}
    >
      <div className="max-w-md mx-auto px-4 py-8">
        <h1 className="text-2xl font-bold text-center mb-8">账户充值</h1>

        {/* Quick amount selection */}
        <div className="mb-6">
          <label
            className={`block text-sm font-medium mb-3 ${isDark ? "text-gray-300" : "text-gray-700"}`}
          >
            选择金额
          </label>
          <div className="grid grid-cols-3 gap-3">
            {QUICK_AMOUNTS.map((val) => (
              <button
                key={val}
                onClick={() => handleSelectAmount(val)}
                className={`py-3 rounded-lg text-lg font-medium transition-all border
                  ${
                    selectedAmount === val && !customAmount
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

        {/* Custom amount */}
        <div className="mb-6">
          <label
            className={`block text-sm font-medium mb-2 ${isDark ? "text-gray-300" : "text-gray-700"}`}
          >
            自定义金额
          </label>
          <div className="relative">
            <span
              className={`absolute left-3 top-1/2 -translate-y-1/2 text-lg ${isDark ? "text-gray-400" : "text-gray-500"}`}
            >
              ¥
            </span>
            <input
              type="number"
              value={customAmount}
              onChange={(e) => handleCustomAmountChange(e.target.value)}
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

        {/* Payment type */}
        <div className="mb-8">
          <label
            className={`block text-sm font-medium mb-3 ${isDark ? "text-gray-300" : "text-gray-700"}`}
          >
            支付方式
          </label>
          <div className="flex gap-3">
            {enabledTypes.includes("wxpay") && (
              <button
                onClick={() => setPaymentType("wxpay")}
                className={`flex-1 py-3 rounded-lg font-medium transition-all border-2 flex items-center justify-center gap-2
                  ${
                    paymentType === "wxpay"
                      ? isDark
                        ? "bg-green-900/30 text-green-400 border-green-500 shadow-md"
                        : "bg-green-50 text-green-700 border-green-500 shadow-md"
                      : isDark
                        ? "bg-gray-800 text-gray-200 border-gray-700 hover:border-green-500"
                        : "bg-white text-gray-700 border-gray-200 hover:border-green-400"
                  }`}
              >
                <img src={wechatIcon} alt="微信支付" className="w-6 h-6 rounded" />
                微信支付
              </button>
            )}
            {enabledTypes.includes("alipay") && (
              <button
                onClick={() => setPaymentType("alipay")}
                className={`flex-1 py-3 rounded-lg font-medium transition-all border-2 flex items-center justify-center gap-2
                  ${
                    paymentType === "alipay"
                      ? isDark
                        ? "bg-blue-900/30 text-blue-400 border-blue-500 shadow-md"
                        : "bg-blue-50 text-blue-700 border-blue-500 shadow-md"
                      : isDark
                        ? "bg-gray-800 text-gray-200 border-gray-700 hover:border-blue-400"
                        : "bg-white text-gray-700 border-gray-200 hover:border-blue-400"
                  }`}
              >
                <img src={alipayIcon} alt="支付宝" className="w-6 h-6 rounded" />
                支付宝
              </button>
            )}
          </div>
        </div>

        {/* Error */}
        {error && (
          <div
            className="mb-4 p-3 rounded-lg bg-red-500/10 border border-red-500/20 text-red-500 text-sm"
            style={{ wordWrap: "break-word" }}
          >
            {error}
          </div>
        )}

        {/* Submit */}
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
            ? "创建订单中..."
            : amount
              ? `支付 ¥${amount.toFixed(2)}`
              : "请选择金额"}
        </button>
      </div>

      {/* QR Code Modal */}
      {showQR && (
        <QRCodeModal
          orderNo={orderNo}
          qrCodeUrl={qrCodeUrl}
          payUrl={payUrl}
          expiresAt={expiresAt}
          paymentType={paymentType}
          isDark={isDark}
          onClose={() => setShowQR(false)}
        />
      )}
    </div>
  );
}

