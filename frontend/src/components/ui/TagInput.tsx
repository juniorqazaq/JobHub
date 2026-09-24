import { X } from "lucide-react";
import { useId, useState, type ClipboardEvent, type KeyboardEvent } from "react";

interface TagInputProps {
  label: string;
  value: string[];
  onChange: (value: string[]) => void;
  placeholder?: string;
  hint?: string;
  removeLabel: (value: string) => string;
}

export function TagInput({ label, value, onChange, placeholder, hint, removeLabel }: TagInputProps) {
  const id = useId();
  const [draft, setDraft] = useState("");

  const add = (candidates: string[]) => {
    const next = [...value];
    const seen = new Set(value.map(normalized));
    for (const candidate of candidates) {
      const item = candidate.trim();
      const key = normalized(item);
      if (!item || seen.has(key)) continue;
      seen.add(key);
      next.push(item);
    }
    onChange(next);
  };
  const commit = () => {
    add(draft.split(","));
    setDraft("");
  };
  const onKeyDown = (event: KeyboardEvent<HTMLInputElement>) => {
    if (event.key === "Enter" || event.key === ",") {
      event.preventDefault();
      commit();
    } else if (event.key === "Backspace" && draft === "" && value.length) {
      onChange(value.slice(0, -1));
    }
  };
  const onPaste = (event: ClipboardEvent<HTMLInputElement>) => {
    const text = event.clipboardData.getData("text");
    if (!text.includes(",")) return;
    event.preventDefault();
    add(text.split(","));
    setDraft("");
  };

  return (
    <div className="ui-field tag-input">
      <label className="ui-label" htmlFor={id}>{label}</label>
      <div className="tag-input__control">
        {value.map((item) => (
          <span className="tag-input__chip" key={normalized(item)}>
            {item}
            <button type="button" aria-label={removeLabel(item)} onMouseDown={(event) => event.preventDefault()} onClick={() => onChange(value.filter((entry) => entry !== item))}>
              <X size={14} aria-hidden="true" />
            </button>
          </span>
        ))}
        <input
          id={id}
          value={draft}
          placeholder={value.length ? undefined : placeholder}
          onChange={(event) => setDraft(event.target.value)}
          onKeyDown={onKeyDown}
          onPaste={onPaste}
          onBlur={commit}
        />
      </div>
      {hint ? <span className="ui-field-hint">{hint}</span> : null}
    </div>
  );
}

function normalized(value: string) {
  return value.trim().toLocaleLowerCase();
}
