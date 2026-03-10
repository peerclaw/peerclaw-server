import i18n from "i18next"
import { initReactI18next } from "react-i18next"
import LanguageDetector from "i18next-browser-languagedetector"
import en from "./locales/en.json"
import zh from "./locales/zh.json"
import es from "./locales/es.json"
import fr from "./locales/fr.json"
import ar from "./locales/ar.json"
import pt from "./locales/pt.json"
import ja from "./locales/ja.json"
import ru from "./locales/ru.json"

i18n
  .use(LanguageDetector)
  .use(initReactI18next)
  .init({
    resources: {
      en: { translation: en },
      zh: { translation: zh },
      es: { translation: es },
      fr: { translation: fr },
      ar: { translation: ar },
      pt: { translation: pt },
      ja: { translation: ja },
      ru: { translation: ru },
    },
    fallbackLng: "en",
    interpolation: { escapeValue: false },
    detection: {
      order: ["localStorage", "navigator"],
      lookupLocalStorage: "peerclaw_lang",
      caches: ["localStorage"],
    },
  })

i18n.on("languageChanged", (lng) => {
  document.documentElement.lang = lng
  document.documentElement.dir = lng === "ar" ? "rtl" : "ltr"
})

// Set initial attributes
document.documentElement.lang = i18n.language
document.documentElement.dir = i18n.language === "ar" ? "rtl" : "ltr"

export default i18n
