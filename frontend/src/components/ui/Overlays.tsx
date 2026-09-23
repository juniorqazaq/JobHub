import { MoreHorizontal, X } from "lucide-react";
import { useEffect, useId, useRef, useState, type ReactNode } from "react";
import { Button, IconButton } from "./Button";

interface DialogProps {
  open: boolean;
  title: string;
  description?: string;
  closeLabel: string;
  onClose: () => void;
  children?: ReactNode;
  footer?: ReactNode;
}

function DialogSurface({ open, title, description, closeLabel, onClose, children, footer, variant }: DialogProps & { variant: "modal" | "drawer" }) {
  const titleId = useId();
  const descriptionId = useId();
  const closeRef = useRef<HTMLButtonElement>(null);

  useEffect(() => {
    if (!open) return;
    closeRef.current?.focus();
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") onClose();
    };
    document.addEventListener("keydown", onKeyDown);
    return () => document.removeEventListener("keydown", onKeyDown);
  }, [open, onClose]);

  if (!open) return null;
  return (
    <div className={`ui-dialog-layer ui-dialog-layer--${variant}`} onMouseDown={(event) => event.target === event.currentTarget && onClose()}>
      <section className={`ui-dialog ui-dialog--${variant}`} role="dialog" aria-modal="true" aria-labelledby={titleId} aria-describedby={description ? descriptionId : undefined}>
        <header>
          <div><h2 id={titleId}>{title}</h2>{description ? <p id={descriptionId}>{description}</p> : null}</div>
          <IconButton ref={closeRef} label={closeLabel} icon={<X size={20} />} variant="quiet" onClick={onClose} />
        </header>
        {children ? <div className="ui-dialog-body">{children}</div> : null}
        {footer ? <footer>{footer}</footer> : null}
      </section>
    </div>
  );
}

export function Modal(props: DialogProps) {
  return <DialogSurface {...props} variant="modal" />;
}

export function Drawer(props: DialogProps) {
  return <DialogSurface {...props} variant="drawer" />;
}

export interface DropdownItem {
  id: string;
  label: string;
  destructive?: boolean;
  onSelect: () => void;
}

export function Dropdown({ label, items }: { label: string; items: DropdownItem[] }) {
  const [open, setOpen] = useState(false);
  return (
    <div className="ui-dropdown">
      <Button variant="secondary" leadingIcon={<MoreHorizontal size={18} />} aria-haspopup="menu" aria-expanded={open} onClick={() => setOpen((value) => !value)}>
        {label}
      </Button>
      {open ? (
        <div className="ui-dropdown-menu" role="menu">
          {items.map((item) => (
            <button key={item.id} type="button" role="menuitem" className={item.destructive ? "is-destructive" : ""} onClick={() => { item.onSelect(); setOpen(false); }}>
              {item.label}
            </button>
          ))}
        </div>
      ) : null}
    </div>
  );
}

export function Tooltip({ label, children }: { label: string; children: ReactNode }) {
  return <span className="ui-tooltip" data-tooltip={label}>{children}</span>;
}
