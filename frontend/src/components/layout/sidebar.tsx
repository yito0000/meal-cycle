import { Link } from "@tanstack/react-router";
import {
  BookOpen,
  Calendar,
  type LucideIcon,
  Refrigerator,
  Settings,
  ShoppingCart,
} from "lucide-react";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { HealthService } from "../../../bindings/github.com/yito0000/meal-cycle";
import type { Health } from "../../../bindings/github.com/yito0000/meal-cycle/models";

type NavItem = {
  to: "/" | "/recipes" | "/fridge" | "/shopping";
  labelKey: string;
  icon: LucideIcon;
  exact?: boolean;
};

const items: NavItem[] = [
  { to: "/", labelKey: "nav.meal_plan", icon: Calendar, exact: true },
  { to: "/recipes", labelKey: "nav.recipes", icon: BookOpen },
  { to: "/fridge", labelKey: "nav.fridge", icon: Refrigerator },
  { to: "/shopping", labelKey: "nav.shopping", icon: ShoppingCart },
];

export function Sidebar() {
  const { t } = useTranslation();
  const [health, setHealth] = useState<Health | null>(null);

  useEffect(() => {
    HealthService.Ping().then(setHealth).catch(console.error);
  }, []);

  return (
    <nav
      aria-label={t("nav.aria_label")}
      className="flex w-52 shrink-0 flex-col gap-1 border-r border-neutral-200 bg-white p-3"
    >
      <div className="mb-2 px-3 py-2 text-sm font-bold">meal-cycle</div>

      {items.map(({ to, labelKey, icon: Icon, exact }) => (
        <Link
          key={to}
          to={to}
          activeOptions={{ exact: exact ?? false }}
          className="flex items-center gap-2 rounded-md px-3 py-2 text-sm text-neutral-800 hover:bg-neutral-100"
          activeProps={{
            className:
              "flex items-center gap-2 rounded-md px-3 py-2 text-sm bg-blue-100 text-blue-900 font-semibold hover:bg-blue-100",
          }}
        >
          <Icon className="h-4 w-4" />
          {t(labelKey)}
        </Link>
      ))}

      <div className="mt-auto flex flex-col gap-2 border-t border-neutral-100 pt-3">
        <div className="px-3 text-xs" title={health?.dbError ?? ""}>
          {health === null ? (
            <span className="text-neutral-400">{t("health.loading")}</span>
          ) : health.dbConnected ? (
            <span className="text-green-700">● {t("health.db_ok_short")}</span>
          ) : (
            <span className="text-red-700">● {t("health.db_error_short")}</span>
          )}
        </div>
        <button
          type="button"
          disabled
          title={t("nav.settings_disabled_hint")}
          className="flex w-full cursor-not-allowed items-center gap-2 rounded-md px-3 py-2 text-sm text-neutral-400"
        >
          <Settings className="h-4 w-4" />
          {t("nav.settings")}
        </button>
      </div>
    </nav>
  );
}
