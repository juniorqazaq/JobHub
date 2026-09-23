import i18n from "i18next";
import { initReactI18next } from "react-i18next";
import { en } from "./locales/en";
import { kk } from "./locales/kk";
import { ru } from "./locales/ru";

const supportedLanguages = new Set(["en", "kk", "ru"]);
const browserLanguage = navigator.language.slice(0, 2);

void i18n.use(initReactI18next).init({
  resources: { en, kk, ru },
  lng: supportedLanguages.has(browserLanguage) ? browserLanguage : "en",
  fallbackLng: "en",
  interpolation: { escapeValue: false },
});

export default i18n;
