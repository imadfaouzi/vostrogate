import { useEffect, useState } from 'react';

// Only in-progress editor drafts belong in localStorage. MySQL remains the
// source of truth for all saved business records and settings.
const prefix = 'vg:draft:';
function recover(key, fallback) {
  try {
    const raw = localStorage.getItem(prefix + key);
    return raw ? JSON.parse(raw) : fallback;
  } catch { return fallback; }
}
export default function useDraft(key, initial) {
  const [draft, setDraft] = useState(() => ({ key, value: recover(key, initial) }));
  // Keep the value associated with its own key. Changing document type cannot
  // accidentally overwrite a different document's draft with stale state.
  const value = draft.key === key ? draft.value : recover(key, initial);
  useEffect(() => {
    if (draft.key !== key) setDraft({ key, value: recover(key, initial) });
  }, [key, draft.key]);
  useEffect(() => {
    if (draft.key !== key) return;
    const timer = setTimeout(() => {
      try { localStorage.setItem(prefix + key, JSON.stringify(draft.value)); } catch {}
    }, 300);
    return () => clearTimeout(timer);
  }, [key, draft]);
  const setValue = (next) => setDraft(previous => {
    const current = previous.key === key ? previous.value : recover(key, initial);
    return { key, value: typeof next === 'function' ? next(current) : next };
  });
  const clear = () => {
    try { localStorage.removeItem(prefix + key); } catch {}
    setDraft({ key, value: initial });
  };
  return [value, setValue, clear];
}
