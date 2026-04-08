import alipayIcon from "../assets/alipay.jpg";
import wechatIcon from "../assets/wechat.png";
import { PAY_MESSAGES } from "./locale";

export const QUICK_AMOUNTS = [20, 50, 100, 200, 400, 600];

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
} as const;
