import { useMemo, useState } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { loadStripe, type Stripe } from "@stripe/stripe-js";
import {
  Elements,
  PaymentElement,
  useStripe,
  useElements,
} from "@stripe/react-stripe-js";

interface Props {
  publishableKey: string;
  clientSecret: string;
  orderNo: string;
  chargeAmount: number; // 实际扣款金额（外币，如美元）
  chargeCurrency: string; // 实际扣款币种，如 usd
  cnyAmount: number; // 对应的人民币充值金额（用于说明到账额度）
  isDark: boolean;
  lang: "zh" | "en";
  onClose: () => void;
}

// loadStripe 应以同一 publishable key 只调用一次，这里按 key 缓存 promise
const stripeCache: Record<string, Promise<Stripe | null>> = {};
function getStripe(pk: string): Promise<Stripe | null> {
  if (!stripeCache[pk]) {
    stripeCache[pk] = loadStripe(pk);
  }
  return stripeCache[pk];
}

// 按币种格式化金额，如 usd → $28.17
function formatMoney(amount: number, currency: string): string {
  try {
    return new Intl.NumberFormat("en-US", {
      style: "currency",
      currency: currency.toUpperCase(),
    }).format(amount);
  } catch {
    return `${currency.toUpperCase()} ${amount.toFixed(2)}`;
  }
}

// 信用卡内嵌支付弹窗：使用 Payment Element 在页面内（Stripe 自有 iframe）渲染卡表单，
// 整体可继续嵌在 sub2api 的 iframe 中，且不强制收集邮箱。
export default function CardCheckout({
  publishableKey,
  clientSecret,
  orderNo,
  chargeAmount,
  chargeCurrency,
  cnyAmount,
  isDark,
  lang,
  onClose,
}: Props) {
  const stripePromise = useMemo(
    () => getStripe(publishableKey),
    [publishableKey],
  );

  return (
    <div
      className="fixed inset-0 z-50 flex items-start justify-center overflow-y-auto bg-black/50 py-8 backdrop-blur-sm"
      onClick={onClose}
    >
      <div
        className={`relative w-[460px] max-w-[94vw] rounded-2xl p-5 shadow-2xl ${
          isDark ? "bg-gray-800" : "bg-white"
        }`}
        onClick={(e) => e.stopPropagation()}
      >
        <button
          onClick={onClose}
          className={`absolute right-3 top-3 flex h-8 w-8 items-center justify-center rounded-full transition-colors ${
            isDark
              ? "text-gray-400 hover:bg-gray-700"
              : "text-gray-500 hover:bg-gray-100"
          }`}
        >
          ✕
        </button>

        <h3
          className={`text-center text-lg font-semibold ${
            isDark ? "text-gray-100" : "text-gray-800"
          }`}
        >
          {lang === "zh" ? "信用卡支付" : "Card Payment"}
        </h3>
        <p
          className={`mb-4 mt-1 text-center text-xs ${
            isDark ? "text-gray-400" : "text-gray-500"
          }`}
        >
          {lang === "zh"
            ? `为账户充值 ¥${cnyAmount.toFixed(2)}`
            : `Top-up ¥${cnyAmount.toFixed(2)} to your account`}
        </p>

        <Elements
          stripe={stripePromise}
          options={{
            clientSecret,
            locale: lang,
            appearance: { theme: isDark ? "night" : "stripe" },
          }}
        >
          <CardForm
            orderNo={orderNo}
            chargeAmount={chargeAmount}
            chargeCurrency={chargeCurrency}
            lang={lang}
            isDark={isDark}
          />
        </Elements>
      </div>
    </div>
  );
}

function CardForm({
  orderNo,
  chargeAmount,
  chargeCurrency,
  lang,
  isDark,
}: {
  orderNo: string;
  chargeAmount: number;
  chargeCurrency: string;
  lang: "zh" | "en";
  isDark: boolean;
}) {
  const stripe = useStripe();
  const elements = useElements();
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState("");

  // 付款成功 → SPA 跳转结果页，保留 token/theme/lang 等参数
  const goResult = () => {
    const params = searchParams.toString();
    navigate(
      `/pay/result?order_no=${orderNo}&status=paid${params ? "&" + params : ""}`,
    );
  };

  const handlePay = async () => {
    if (!stripe || !elements) return;
    setSubmitting(true);
    setError("");

    // 部分 3DS 卡需要跳转验证时才用到 return_url；redirect: "if_required" 下普通卡不跳转
    const returnUrl = `${window.location.origin}/pay/result?order_no=${orderNo}&status=paid`;
    const { error: confirmErr, paymentIntent } = await stripe.confirmPayment({
      elements,
      confirmParams: { return_url: returnUrl },
      redirect: "if_required",
    });

    if (confirmErr) {
      setError(
        confirmErr.message || (lang === "zh" ? "支付失败，请重试" : "Payment failed"),
      );
      setSubmitting(false);
      return;
    }

    if (
      paymentIntent &&
      (paymentIntent.status === "succeeded" ||
        paymentIntent.status === "processing")
    ) {
      goResult();
      return;
    }

    setError(lang === "zh" ? "支付未完成，请重试" : "Payment not completed");
    setSubmitting(false);
  };

  return (
    <form
      onSubmit={(e) => {
        e.preventDefault();
        handlePay();
      }}
    >
      <PaymentElement options={{ layout: "tabs" }} />

      {error && (
        <div
          className="mt-3 rounded-lg border border-red-500/20 bg-red-500/10 p-2 text-sm text-red-500"
          style={{ wordWrap: "break-word" }}
        >
          {error}
        </div>
      )}

      <button
        type="submit"
        disabled={!stripe || submitting}
        className={`mt-4 w-full rounded-full py-3 text-base font-bold transition-all ${
          !stripe || submitting
            ? "cursor-not-allowed bg-slate-300 text-slate-100"
            : isDark
              ? "bg-sky-500 text-white hover:bg-sky-400"
              : "bg-sky-600 text-white hover:bg-sky-500"
        }`}
      >
        {submitting
          ? lang === "zh"
            ? "支付中..."
            : "Processing..."
          : lang === "zh"
            ? `支付 ${formatMoney(chargeAmount, chargeCurrency)}`
            : `Pay ${formatMoney(chargeAmount, chargeCurrency)}`}
      </button>
    </form>
  );
}
