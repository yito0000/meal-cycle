import { Fragment } from "react";
import { useTranslation } from "react-i18next";
import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from "@/components/ui/breadcrumb";

export type BreadcrumbEntry = { label: string; onSelect?: () => void };

// memory history では href が意味を持たず、S04 では移る前に未保存の確認を挟むので、項目は Link ではなく button にする (D10)。
export function PageBreadcrumb({ items }: { items: BreadcrumbEntry[] }) {
  const { t } = useTranslation();
  return (
    <Breadcrumb aria-label={t("breadcrumb.aria_label")}>
      <BreadcrumbList>
        {items.map((item, i) => {
          const last = i === items.length - 1;
          return (
            // biome-ignore lint/suspicious/noArrayIndexKey: 項目は位置で意味が決まり、並びは固定。
            <Fragment key={i}>
              <BreadcrumbItem>
                {last || !item.onSelect ? (
                  <BreadcrumbPage aria-current={last ? "page" : undefined}>
                    {item.label}
                  </BreadcrumbPage>
                ) : (
                  <button
                    type="button"
                    onClick={item.onSelect}
                    className="transition-colors hover:text-neutral-900 hover:underline"
                  >
                    {item.label}
                  </button>
                )}
              </BreadcrumbItem>
              {!last && <BreadcrumbSeparator />}
            </Fragment>
          );
        })}
      </BreadcrumbList>
    </Breadcrumb>
  );
}
