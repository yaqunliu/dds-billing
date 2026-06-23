import type { AppLang } from "./i18n";

export const STATUS_LABELS: Record<AppLang, Record<string, string>> = {
  zh: {
    pending: "待支付",
    paid: "已支付",
    recharging: "充值中",
    completed: "已完成",
    failed: "失败",
    expired: "已过期",
  },
  en: {
    pending: "Pending",
    paid: "Paid",
    recharging: "Recharging",
    completed: "Completed",
    failed: "Failed",
    expired: "Expired",
  },
};

export type PolicySection = {
  heading?: string;
  lines?: string[];
  bullets?: string[];
};

type PayMessages = {
  title: string;
  amount: string;
  customAmount: string;
  paymentType: string;
  wechatPay: string;
  alipay: string;
  creditCard: string;
  cardMinTip: (min: number) => string;
  redirectingToStripe: string;
  missingToken: string;
  createOrderFailed: string;
  networkError: string;
  creatingOrder: string;
  payAmount: (value: number) => string;
  selectAmount: string;
  amountRange: (min: number, max: number) => string;
  policyTitle: string;
  policySubtitle: string;
  policySections: (baseMin: number, cardMin: number) => PolicySection[];
};

export const PAY_MESSAGES = {
  zh: {
    title: "账户充值",
    amount: "选择金额",
    customAmount: "自定义金额",
    paymentType: "支付方式",
    wechatPay: "微信支付",
    alipay: "支付宝",
    creditCard: "信用卡",
    cardMinTip: (min: number) => `信用卡支付最低 ¥${min} 起充`,
    redirectingToStripe: "正在跳转到安全支付页面...",
    missingToken: "缺少用户凭证，请从平台入口进入",
    createOrderFailed: "创建订单失败",
    networkError: "网络错误，请重试",
    creatingOrder: "创建订单中...",
    payAmount: (value: number) => `支付 ¥${value.toFixed(2)}`,
    selectAmount: "请选择金额",
    amountRange: (min: number, max: number) =>
      `请输入 ${min} - ${max} 之间的金额`,
    policyTitle: "购买规则说明",
    policySubtitle: "充值与退款说明",
    policySections: (baseMin: number, cardMin: number) => [
      {
        heading: "支持方式",
        bullets: ["微信", "支付宝", "信用卡（Stripe）"],
      },
      {
        heading: "微信 / 支付宝",
        lines: [`最低充值 ${baseMin} 元`],
      },
      {
        heading: "信用卡（Stripe）",
        lines: [`最低充值 ${cardMin} 元`],
      },
      {
        lines: [
          "微信 / 支付宝按 1 元 = 1 平台代币 到账；信用卡按 Stripe 实际美元结算，到账金额以充值时选择的平台代币数量为准。",
        ],
      },
      {
        heading: "到账说明",
        lines: [
          "通常即时到账（最长不超过 5 分钟）。",
          "如遇到账异常，请联系管理员协助处理。",
        ],
      },
      {
        heading: "退款规则",
        lines: [
          "退款金额 = 充值金额 − 已消费金额 − 支付手续费",
          "手续费 = 充值金额 × 5%",
        ],
        bullets: [
          "退款将原路退回至支付账户。",
          "到账时间通常为 1–3 个工作日。",
        ],
      },
      {
        heading: "政策说明",
        lines: [
          "完成充值即视为您已阅读并同意本充值与退款政策。",
          "本平台保留对本政策的最终解释权。",
        ],
      },
    ],
  },
  en: {
    title: "Account Recharge",
    amount: "Amount",
    customAmount: "Custom Amount",
    paymentType: "Payment Method",
    wechatPay: "WeChat Pay",
    alipay: "Alipay",
    creditCard: "Credit Card",
    cardMinTip: (min: number) => `Card payment requires a minimum of ¥${min}`,
    redirectingToStripe: "Redirecting to secure checkout...",
    missingToken: "Missing user credentials. Please enter from the platform.",
    createOrderFailed: "Failed to create order",
    networkError: "Network error, please try again",
    creatingOrder: "Creating order...",
    payAmount: (value: number) => `Pay ¥${value.toFixed(2)}`,
    selectAmount: "Please select an amount",
    amountRange: (min: number, max: number) =>
      `Please enter an amount between ${min} and ${max}`,
    policyTitle: "Purchase Rules",
    policySubtitle: "Recharge & Refund Policy",
    policySections: (baseMin: number, cardMin: number) => [
      {
        heading: "Supported Methods",
        bullets: ["WeChat Pay", "Alipay", "Credit Card (Stripe)"],
      },
      {
        heading: "WeChat / Alipay",
        lines: [`Minimum recharge ¥${baseMin}`],
      },
      {
        heading: "Credit Card (Stripe)",
        lines: [`Minimum recharge ¥${cardMin}`],
      },
      {
        lines: [
          "WeChat / Alipay is credited at ¥1 = 1 platform token; credit card is settled in actual USD via Stripe, and the credited amount is based on the number of platform tokens selected at recharge.",
        ],
      },
      {
        heading: "Crediting",
        lines: [
          "Usually credited instantly (within 5 minutes at most).",
          "If crediting appears abnormal, please contact the administrator for assistance.",
        ],
      },
      {
        heading: "Refund Rules",
        lines: [
          "Refund amount = recharge amount − consumed amount − payment processing fee",
          "Processing fee = recharge amount × 5%",
        ],
        bullets: [
          "Refunds are returned to the original payment account.",
          "Crediting usually takes 1–3 business days.",
        ],
      },
      {
        heading: "Policy Notice",
        lines: [
          "Completing a recharge means you have read and agreed to this Recharge & Refund Policy.",
          "The platform reserves the right of final interpretation of this policy.",
        ],
      },
    ],
  },
} satisfies Record<AppLang, PayMessages>;

export const ORDERS_MESSAGES = {
  zh: {
    title: "充值记录",
    loading: "加载中...",
    empty: "暂无充值记录",
    prev: "上一页",
    next: "下一页",
  },
  en: {
    title: "Recharge Records",
    loading: "Loading...",
    empty: "No recharge records yet",
    prev: "Previous",
    next: "Next",
  },
} satisfies Record<AppLang, Record<string, string>>;

export const RESULT_MESSAGES = {
  zh: {
    loading: "加载中...",
    rechargeSuccess: "充值成功",
    rechargeFailed: "充值失败",
    orderNo: "订单号",
    amount: "金额",
    status: "状态",
    paidAt: "支付时间",
    continueRecharge: "继续充值",
  },
  en: {
    loading: "Loading...",
    rechargeSuccess: "Recharge Successful",
    rechargeFailed: "Recharge Failed",
    orderNo: "Order No.",
    amount: "Amount",
    status: "Status",
    paidAt: "Paid At",
    continueRecharge: "Continue Recharge",
  },
} satisfies Record<AppLang, Record<string, string>>;

export const QRCODE_MESSAGES = {
  zh: {
    paymentLabels: {
      wxpay: "微信",
      alipay: "支付宝",
    },
    scanTitle: (paymentLabel: string) => `${paymentLabel}扫码支付`,
    qrAlt: "支付二维码",
    qrLoading: "二维码加载中...",
    countdownText: (time: string) => `请在 ${time} 内完成支付`,
    orderNo: "订单号",
    paymentSuccess: "支付成功",
    recharging: "正在充值到账...",
    rechargeComplete: "充值完成",
    redirecting: "即将跳转...",
    orderExpired: "订单已过期",
    placeOrderAgain: "重新下单",
    rechargeFailed: "充值失败",
    supportTip: "如已扣款，请联系客服处理",
    back: "返回",
  },
  en: {
    paymentLabels: {
      wxpay: "WeChat",
      alipay: "Alipay",
    },
    scanTitle: (paymentLabel: string) => `${paymentLabel} Scan to Pay`,
    qrAlt: "Payment QR code",
    qrLoading: "Loading QR code...",
    countdownText: (time: string) => `Please complete payment within ${time}`,
    orderNo: "Order No.",
    paymentSuccess: "Payment Successful",
    recharging: "Recharging your balance...",
    rechargeComplete: "Recharge Completed",
    redirecting: "Redirecting...",
    orderExpired: "Order Expired",
    placeOrderAgain: "Place New Order",
    rechargeFailed: "Recharge Failed",
    supportTip: "If you were charged, please contact support.",
    back: "Back",
  },
} satisfies Record<
  AppLang,
  {
    paymentLabels: Record<"wxpay" | "alipay", string>;
    scanTitle: (paymentLabel: string) => string;
    qrAlt: string;
    qrLoading: string;
    countdownText: (time: string) => string;
    orderNo: string;
    paymentSuccess: string;
    recharging: string;
    rechargeComplete: string;
    redirecting: string;
    orderExpired: string;
    placeOrderAgain: string;
    rechargeFailed: string;
    supportTip: string;
    back: string;
  }
>;

export function pickLocale<T>(messages: Record<AppLang, T>, lang: AppLang): T {
  return messages[lang];
}
