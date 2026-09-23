import { ChevronLeft, ChevronRight, X } from "lucide-react";
import type { ReactNode } from "react";
import { IconButton } from "./Button";

export function Chip({ children, onRemove, removeLabel }: { children: ReactNode; onRemove?: () => void; removeLabel?: string }) {
  return (
    <span className="ui-chip">
      {children}
      {onRemove ? <IconButton variant="quiet" label={removeLabel ?? "Remove"} icon={<X size={14} />} onClick={onRemove} /> : null}
    </span>
  );
}

export interface TabItem { id: string; label: string }

export function Tabs({ tabs, activeId, onChange, label }: { tabs: TabItem[]; activeId: string; onChange: (id: string) => void; label: string }) {
  return (
    <div className="ui-tabs" role="tablist" aria-label={label}>
      {tabs.map((tab) => (
        <button key={tab.id} type="button" role="tab" aria-selected={activeId === tab.id} onClick={() => onChange(tab.id)}>
          {tab.label}
        </button>
      ))}
    </div>
  );
}

export function Pagination({ page, totalPages, onChange, label, previousLabel, nextLabel, pageLabel }: { page: number; totalPages: number; onChange: (page: number) => void; label: string; previousLabel: string; nextLabel: string; pageLabel: (page: number) => string }) {
  const pages = Array.from({ length: totalPages }, (_, index) => index + 1);
  return (
    <nav className="ui-pagination" aria-label={label}>
      <IconButton label={previousLabel} icon={<ChevronLeft size={18} />} disabled={page <= 1} onClick={() => onChange(page - 1)} />
      {pages.map((item) => (
        <button key={item} type="button" aria-label={pageLabel(item)} aria-current={item === page ? "page" : undefined} onClick={() => onChange(item)}>{item}</button>
      ))}
      <IconButton label={nextLabel} icon={<ChevronRight size={18} />} disabled={page >= totalPages} onClick={() => onChange(page + 1)} />
    </nav>
  );
}
