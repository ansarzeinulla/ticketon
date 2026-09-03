import type { Locale } from "./config";

/**
 * The message catalogues for the customer-facing pages.
 *
 * English is the source of truth for the *shape*: its object defines the
 * `Dictionary` type, so every other locale is checked at compile time to have
 * exactly the same keys - a missing Kazakh string is a type error, not a blank
 * space a visitor finds at runtime.
 *
 * Interpolation is `{name}` placeholders, filled by `translate` below. Money
 * and dates are never interpolated as words here: they are formatted by
 * `money.ts` and `datetime.ts`, which already localise them.
 */
const en = {
  header: {
    organizers: "Organizers",
    language: "Language",
  },
  catalogue: {
    title: "What's on",
    subtitle: "Public events with tickets available on BiletFlow, soonest first.",
    unavailable: "The events list is unavailable right now. Please try again shortly.",
    emptyFirst: "No public events are on sale at the moment.",
    emptyMore: "Nothing on this page.",
    earlier: "← Earlier",
    later: "Later →",
    pageInfo: "Page {page} of {last} · {total} {events}",
    eventOne: "event",
    eventMany: "events",
  },
  eventPage: {
    when: "When",
    until: "until {time} ({tz})",
    where: "Where",
    suspendedTitle: "Ticket sales suspended",
    suspendedBody:
      "BiletFlow has suspended this event pending review, so no new tickets can be bought. Tickets already issued remain valid — if you are holding one, contact the organizer from your order page for the latest information.",
    paidPendingTitle: "Paid tickets are not on sale yet",
    paidPendingBody: "The organizer is still completing BiletFlow's activation checks for this event.",
    paidPendingFree: " Free tickets below can be booked in the meantime.",
    paidPendingCheckBack: " Check back shortly.",
    refundPolicy: "Refund policy",
  },
  tickets: {
    heading: "Tickets",
    none: "The organizer has not published any ticket types for this event yet.",
    soldOutTitle: "Sold out",
    soldOutBody: "Every ticket for this event has been taken.",
    notOnSaleTitle: "Not on sale",
    notOnSaleBody: "Tickets for this event are not currently available.",
    free: "Free",
    soldOutShort: "Sold out",
    remaining: "{remaining} of {total} left",
    addOne: "Add one {name}",
    removeOne: "Remove one {name}",
    noneSelected: "No tickets selected",
    countOne: "{count} ticket",
    countMany: "{count} tickets",
    getTickets: "Get tickets",
  },
  checkout: {
    title: "Checkout",
    promo: "Promo {code}",
    total: "Total",
    nameLabel: "Full name",
    namePlaceholder: "Nurlan Amanov",
    nameRequired: "Your name is required.",
    emailLabel: "Email",
    emailPlaceholder: "nurlan@example.kz",
    emailHint: "Your tickets are issued to this address.",
    emailRequired: "Your email is required.",
    simulatedNote:
      "Simulated payment — no card is charged and no money moves. This is a demonstration checkout.",
    processing: "Processing…",
    pay: "Pay {amount} (simulated)",
    cancel: "Cancel",
    correctFields: "Please correct the highlighted fields.",
    failed: "The payment simulation failed. Please try again.",
    priceUpdated: " The price has been updated.",
    soldOutDuring:
      "These tickets sold out while you were choosing. The page has been refreshed.",
    soldOutPartial: "{message} The page has been refreshed with what is left.",
  },
  order: {
    metaTitle: "Order confirmation",
    confirmed: "Order confirmed",
    issuedOne: "{count} ticket issued to {email}.",
    issuedMany: "{count} tickets issued to {email}.",
    orderNumber: "Order number",
    orderId: "Order ID",
    status: "Status",
    totalPaid: "Total paid",
    summary: "Order summary",
    yourTickets: "Your tickets",
    ticketsHint:
      "Show the QR code at the entrance, or download the A4 PDF to print. Each ticket admits one person once.",
    simulatedNote: "This was a simulated payment. No card was charged and no money moved.",
    back: "Back to BiletFlow",
  },
  promo: {
    haveCode: "Have a promo code?",
    apply: "Apply",
    checking: "Checking the code…",
    applied: "Promo code {code} applied",
    percentOff: "{value}% off",
    amountOff: "{amount} off",
    selectedOnly: "applies to selected ticket types only",
    remove: "Remove",
    subtotal: "Subtotal",
    discount: "Discount",
    newTotal: "New total",
    chooseFirst: "Choose your tickets first, then the discount can be calculated.",
    couldNotApply: "That code could not be applied.",
  },
  ticketCard: {
    admissionCode: "Admission code",
    download: "Download PDF ticket",
    qrAlt: "Admission QR code for ticket {code}",
    statusValid: "valid",
    statusCheckedIn: "checked in",
    statusCancelled: "cancelled",
    statusRefunded: "refunded",
  },
};

/** The shape every locale must satisfy, taken from English. */
export type Dictionary = typeof en;

/** A dotted path into the dictionary, e.g. "checkout.title". */
type Join<K, P> = K extends string
  ? P extends string
    ? `${K}${"" extends P ? "" : "."}${P}`
    : never
  : never;

type Leaves<T> = T extends object
  ? { [K in keyof T]-?: Join<K & string, Leaves<T[K]>> }[keyof T]
  : "";

export type MessageKey = Leaves<Dictionary>;

const ru: Dictionary = {
  header: {
    organizers: "Организаторам",
    language: "Язык",
  },
  catalogue: {
    title: "Афиша",
    subtitle: "Публичные события с билетами на BiletFlow, ближайшие первыми.",
    unavailable: "Список событий сейчас недоступен. Пожалуйста, попробуйте чуть позже.",
    emptyFirst: "Сейчас нет публичных событий в продаже.",
    emptyMore: "На этой странице ничего нет.",
    earlier: "← Раньше",
    later: "Позже →",
    pageInfo: "Страница {page} из {last} · {total} {events}",
    eventOne: "событие",
    eventMany: "событий",
  },
  eventPage: {
    when: "Когда",
    until: "до {time} ({tz})",
    where: "Где",
    suspendedTitle: "Продажа билетов приостановлена",
    suspendedBody:
      "BiletFlow приостановил это событие до проверки, поэтому новые билеты купить нельзя. Уже выпущенные билеты остаются действительными — если у вас есть билет, свяжитесь с организатором со страницы заказа, чтобы узнать актуальную информацию.",
    paidPendingTitle: "Платные билеты ещё не в продаже",
    paidPendingBody: "Организатор ещё проходит проверки активации BiletFlow для этого события.",
    paidPendingFree: " Бесплатные билеты ниже можно забронировать уже сейчас.",
    paidPendingCheckBack: " Загляните чуть позже.",
    refundPolicy: "Условия возврата",
  },
  tickets: {
    heading: "Билеты",
    none: "Организатор ещё не опубликовал ни одного типа билетов для этого события.",
    soldOutTitle: "Всё продано",
    soldOutBody: "Все билеты на это событие раскуплены.",
    notOnSaleTitle: "Не в продаже",
    notOnSaleBody: "Билеты на это событие сейчас недоступны.",
    free: "Бесплатно",
    soldOutShort: "Продано",
    remaining: "осталось {remaining} из {total}",
    addOne: "Добавить один: {name}",
    removeOne: "Убрать один: {name}",
    noneSelected: "Билеты не выбраны",
    countOne: "{count} билет",
    countMany: "{count} билетов",
    getTickets: "Получить билеты",
  },
  checkout: {
    title: "Оформление",
    promo: "Промокод {code}",
    total: "Итого",
    nameLabel: "Полное имя",
    namePlaceholder: "Нурлан Аманов",
    nameRequired: "Укажите ваше имя.",
    emailLabel: "Эл. почта",
    emailPlaceholder: "nurlan@example.kz",
    emailHint: "Билеты будут выпущены на этот адрес.",
    emailRequired: "Укажите вашу эл. почту.",
    simulatedNote:
      "Симуляция оплаты — карта не списывается и деньги не двигаются. Это демонстрационное оформление.",
    processing: "Обработка…",
    pay: "Оплатить {amount} (симуляция)",
    cancel: "Отмена",
    correctFields: "Пожалуйста, исправьте выделенные поля.",
    failed: "Симуляция оплаты не удалась. Пожалуйста, попробуйте снова.",
    priceUpdated: " Цена обновлена.",
    soldOutDuring: "Эти билеты раскупили, пока вы выбирали. Страница обновлена.",
    soldOutPartial: "{message} Страница обновлена с учётом оставшихся билетов.",
  },
  order: {
    metaTitle: "Подтверждение заказа",
    confirmed: "Заказ подтверждён",
    issuedOne: "{count} билет выпущен для {email}.",
    issuedMany: "{count} билетов выпущено для {email}.",
    orderNumber: "Номер заказа",
    orderId: "ID заказа",
    status: "Статус",
    totalPaid: "Итого оплачено",
    summary: "Состав заказа",
    yourTickets: "Ваши билеты",
    ticketsHint:
      "Покажите QR-код на входе или скачайте PDF формата A4 для печати. Каждый билет пропускает одного человека один раз.",
    simulatedNote: "Это была симуляция оплаты. Карта не списывалась и деньги не двигались.",
    back: "Назад в BiletFlow",
  },
  promo: {
    haveCode: "Есть промокод?",
    apply: "Применить",
    checking: "Проверяем код…",
    applied: "Промокод {code} применён",
    percentOff: "скидка {value}%",
    amountOff: "скидка {amount}",
    selectedOnly: "действует только для выбранных типов билетов",
    remove: "Убрать",
    subtotal: "Подытог",
    discount: "Скидка",
    newTotal: "Новый итог",
    chooseFirst: "Сначала выберите билеты, затем можно рассчитать скидку.",
    couldNotApply: "Этот код не удалось применить.",
  },
  ticketCard: {
    admissionCode: "Код входа",
    download: "Скачать PDF-билет",
    qrAlt: "QR-код для входа по билету {code}",
    statusValid: "действителен",
    statusCheckedIn: "отмечен на входе",
    statusCancelled: "отменён",
    statusRefunded: "возвращён",
  },
};

const kk: Dictionary = {
  header: {
    organizers: "Ұйымдастырушыларға",
    language: "Тіл",
  },
  catalogue: {
    title: "Афиша",
    subtitle: "BiletFlow-дағы билеті бар ашық іс-шаралар, ең жақындары бірінші.",
    unavailable: "Іс-шаралар тізімі қазір қолжетімсіз. Сәл кейінірек қайталап көріңіз.",
    emptyFirst: "Қазір сатылымда ашық іс-шаралар жоқ.",
    emptyMore: "Бұл бетте ештеңе жоқ.",
    earlier: "← Ерте",
    later: "Кейін →",
    pageInfo: "{last} беттің {page}-беті · {total} {events}",
    eventOne: "іс-шара",
    eventMany: "іс-шара",
  },
  eventPage: {
    when: "Қашан",
    until: "{time} дейін ({tz})",
    where: "Қайда",
    suspendedTitle: "Билет сатылымы тоқтатылды",
    suspendedBody:
      "BiletFlow бұл іс-шараны тексеруге дейін тоқтатты, сондықтан жаңа билет сатып алуға болмайды. Бұрын берілген билеттер жарамды болып қала береді — билетіңіз болса, соңғы ақпаратты білу үшін тапсырыс бетінен ұйымдастырушыға хабарласыңыз.",
    paidPendingTitle: "Ақылы билеттер әлі сатылымда емес",
    paidPendingBody: "Ұйымдастырушы осы іс-шара үшін BiletFlow белсендіру тексерулерін әлі аяқтап жатыр.",
    paidPendingFree: " Төмендегі тегін билеттерді осы кезде де брондауға болады.",
    paidPendingCheckBack: " Сәл кейінірек қайта қараңыз.",
    refundPolicy: "Қайтару шарттары",
  },
  tickets: {
    heading: "Билеттер",
    none: "Ұйымдастырушы бұл іс-шара үшін әлі бірде-бір билет түрін жарияламаған.",
    soldOutTitle: "Барлығы сатылды",
    soldOutBody: "Бұл іс-шараның барлық билеті сатылып кетті.",
    notOnSaleTitle: "Сатылымда емес",
    notOnSaleBody: "Бұл іс-шараның билеттері қазір қолжетімсіз.",
    free: "Тегін",
    soldOutShort: "Сатылды",
    remaining: "{total} ішінен {remaining} қалды",
    addOne: "Бір {name} қосу",
    removeOne: "Бір {name} алып тастау",
    noneSelected: "Билет таңдалмаған",
    countOne: "{count} билет",
    countMany: "{count} билет",
    getTickets: "Билет алу",
  },
  checkout: {
    title: "Рәсімдеу",
    promo: "Промокод {code}",
    total: "Барлығы",
    nameLabel: "Толық аты-жөні",
    namePlaceholder: "Нұрлан Аманов",
    nameRequired: "Атыңызды көрсетіңіз.",
    emailLabel: "Электрондық пошта",
    emailPlaceholder: "nurlan@example.kz",
    emailHint: "Билеттер осы мекенжайға беріледі.",
    emailRequired: "Электрондық поштаңызды көрсетіңіз.",
    simulatedNote:
      "Төлем симуляциясы — картадан ақша алынбайды және қаражат аударылмайды. Бұл көрсетілім рәсімдеуі.",
    processing: "Өңделуде…",
    pay: "{amount} төлеу (симуляция)",
    cancel: "Бас тарту",
    correctFields: "Белгіленген өрістерді түзетіңіз.",
    failed: "Төлем симуляциясы сәтсіз аяқталды. Қайталап көріңіз.",
    priceUpdated: " Баға жаңартылды.",
    soldOutDuring: "Сіз таңдап жатқанда бұл билеттер сатылып кетті. Бет жаңартылды.",
    soldOutPartial: "{message} Бет қалған билеттермен жаңартылды.",
  },
  order: {
    metaTitle: "Тапсырысты растау",
    confirmed: "Тапсырыс расталды",
    issuedOne: "{email} үшін {count} билет берілді.",
    issuedMany: "{email} үшін {count} билет берілді.",
    orderNumber: "Тапсырыс нөмірі",
    orderId: "Тапсырыс ID",
    status: "Мәртебесі",
    totalPaid: "Төленген сома",
    summary: "Тапсырыс құрамы",
    yourTickets: "Сіздің билеттеріңіз",
    ticketsHint:
      "Кіреберісте QR-кодты көрсетіңіз немесе басып шығару үшін A4 PDF жүктеп алыңыз. Әр билет бір адамды бір рет өткізеді.",
    simulatedNote: "Бұл төлем симуляциясы болды. Картадан ақша алынбады және қаражат аударылмады.",
    back: "BiletFlow-ға оралу",
  },
  promo: {
    haveCode: "Промокод бар ма?",
    apply: "Қолдану",
    checking: "Код тексерілуде…",
    applied: "{code} промокоды қолданылды",
    percentOff: "{value}% жеңілдік",
    amountOff: "{amount} жеңілдік",
    selectedOnly: "тек таңдалған билет түрлеріне қолданылады",
    remove: "Алып тастау",
    subtotal: "Аралық сома",
    discount: "Жеңілдік",
    newTotal: "Жаңа сома",
    chooseFirst: "Алдымен билеттерді таңдаңыз, содан кейін жеңілдік есептеледі.",
    couldNotApply: "Бұл кодты қолдану мүмкін болмады.",
  },
  ticketCard: {
    admissionCode: "Кіру коды",
    download: "PDF билетті жүктеу",
    qrAlt: "{code} билеті бойынша кіру QR-коды",
    statusValid: "жарамды",
    statusCheckedIn: "тіркелген",
    statusCancelled: "тоқтатылған",
    statusRefunded: "қайтарылған",
  },
};

const dictionaries: Record<Locale, Dictionary> = { kk, ru, en };

export function getDictionaryFor(locale: Locale): Dictionary {
  return dictionaries[locale];
}
