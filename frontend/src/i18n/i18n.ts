import i18n from "i18next";
import { initReactI18next } from "react-i18next";
import { kk } from "./locales/kk";
import { ru } from "./locales/ru";

const defaultLanguage = "kk";
const languageStorageKey = "i18nextLng";
const supportedLanguages = new Set(["kk", "ru"]);
const browserLanguage = navigator.language.slice(0, 2);
const storedLanguage = readStoredLanguage();

function supportedLanguage(language: string | null | undefined) {
  const normalized = language?.slice(0, 2);
  return normalized && supportedLanguages.has(normalized) ? normalized : defaultLanguage;
}

const initialLanguage = supportedLanguage(storedLanguage ?? browserLanguage);

if (storedLanguage && storedLanguage !== initialLanguage) {
  storeLanguage(initialLanguage);
}

void i18n.use(initReactI18next).init({
  resources: { kk, ru },
  lng: initialLanguage,
  fallbackLng: defaultLanguage,
  supportedLngs: Array.from(supportedLanguages),
  interpolation: { escapeValue: false },
});

i18n.on("languageChanged", (language) => {
  storeLanguage(supportedLanguage(language));
});

export default i18n;

function readStoredLanguage() {
  try {
    return window.localStorage.getItem(languageStorageKey);
  } catch {
    return null;
  }
}

function storeLanguage(language: string) {
  try {
    window.localStorage.setItem(languageStorageKey, language);
  } catch {
    // Some embedded browsers disable storage; i18n should still render with the in-memory locale.
  }
}
