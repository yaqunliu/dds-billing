import alipayIcon from "../assets/alipay.jpg";
import wechatIcon from "../assets/wechat.png";
import cardIcon from "../assets/card.svg";
import { PAY_MESSAGES } from "./locale";

// 非信用卡支付方式的快捷金额
export const QUICK_AMOUNTS = [30, 50, 100, 200, 400, 600];

// 信用卡专属快捷金额（最低 200 起充）
export const CARD_QUICK_AMOUNTS = [200, 300, 500, 1000, 2000, 3000];

export const PAYMENT_TYPE_CONFIG = {
  wxpay: {
    icon: wechatIcon,
    getLabel: (t: typeof PAY_MESSAGES.zh) => t.wechatPay,
    activeClass: "bg-green-900/30 text-green-400 border-green-500 shadow-md",
    activeLightClass: "bg-green-50 text-green-700 border-green-500 shadow-md",
    inactiveHoverClass: "hover:border-green-500",
    inactiveLightHoverClass: "hover:border-green-400",
  },
  alipay: {
    icon: alipayIcon,
    getLabel: (t: typeof PAY_MESSAGES.zh) => t.alipay,
    activeClass: "bg-blue-900/30 text-blue-400 border-blue-500 shadow-md",
    activeLightClass: "bg-blue-50 text-blue-700 border-blue-500 shadow-md",
    inactiveHoverClass: "hover:border-blue-400",
    inactiveLightHoverClass: "hover:border-blue-400",
  },
  card: {
    icon: cardIcon,
    getLabel: (t: typeof PAY_MESSAGES.zh) => t.creditCard,
    activeClass: "bg-indigo-900/30 text-indigo-300 border-indigo-500 shadow-md",
    activeLightClass: "bg-indigo-50 text-indigo-700 border-indigo-500 shadow-md",
    inactiveHoverClass: "hover:border-indigo-400",
    inactiveLightHoverClass: "hover:border-indigo-400",
  },
} as const;
