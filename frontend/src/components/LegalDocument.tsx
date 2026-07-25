import { BrandMark } from './BrandMark'

export type LegalDocumentType = 'privacy' | 'terms'
type LegalLanguage = 'en' | 'ru'

type LegalSection = {
  title: string
  text: string
}

type LegalPage = {
  title: string
  updated: string
  intro: string
  sections: LegalSection[]
}

const legalContent: Record<LegalLanguage, {
  close: string
  product: string
  privacy: LegalPage
  terms: LegalPage
}> = {
  en: {
    close: 'Close',
    product: 'Sparx',
    privacy: {
      title: 'Privacy Policy',
      updated: 'Effective July 25, 2026',
      intro: 'Sparx is a private progress tracker for a small group of users. This policy explains which data is needed to operate the service.',
      sections: [
        {
          title: 'Data we store',
          text: 'Account email, display name and timezone; goals, sessions, notes, tags and progress statistics; authentication sessions, push subscriptions and limited security records. Passwords are stored only as protected hashes and cannot be read back.',
        },
        {
          title: 'How data is used',
          text: 'Data is used to authenticate you, calculate progress and streaks, provide reminders, protect accounts and maintain the service. It is not sold, used for advertising or used to build marketing profiles.',
        },
        {
          title: 'Service providers',
          text: 'Hosting, email and notification providers may process only the data required to deliver those functions. Sparx does not intentionally share learning history with other users or unrelated third parties.',
        },
        {
          title: 'Control and retention',
          text: 'Your data remains while your account is active. You can export it or permanently delete the account in Settings. Short-lived security records are removed automatically after their retention period.',
        },
        {
          title: 'Keeping data safe',
          text: 'Reasonable technical safeguards are used, but no beta service can guarantee absolute security or uninterrupted availability. Avoid putting highly sensitive personal information in notes.',
        },
      ],
    },
    terms: {
      title: 'Terms of Use',
      updated: 'Effective July 25, 2026',
      intro: 'Sparx is a non-commercial beta service for tracking personal learning and practice goals.',
      sections: [
        {
          title: 'Your account',
          text: 'Keep your credentials private and provide an email address you can access. You are responsible for activity performed through your account.',
        },
        {
          title: 'Acceptable use',
          text: 'Do not attempt to bypass security controls, access another user’s data, overload the service or use it for unlawful activity.',
        },
        {
          title: 'Your content',
          text: 'You keep ownership of your goals, notes and session history. You grant the service only the technical permission required to store and display that information for you.',
        },
        {
          title: 'Availability and backups',
          text: 'The service is provided as a beta without a guarantee of continuous availability. Important information should also be kept in an export under your control.',
        },
        {
          title: 'Leaving the service',
          text: 'You can delete your account and its stored data in Settings. Access may be restricted when an account is used to harm the service or other users.',
        },
      ],
    },
  },
  ru: {
    close: 'Закрыть',
    product: 'Sparx',
    privacy: {
      title: 'Политика конфиденциальности',
      updated: 'Действует с 25 июля 2026 года',
      intro: 'Sparx — закрытый трекер прогресса для небольшой группы пользователей. Здесь описано, какие данные нужны для работы сервиса.',
      sections: [
        {
          title: 'Какие данные хранятся',
          text: 'Email, отображаемое имя и часовой пояс; цели, сессии, заметки, теги и статистика; сессии авторизации, push-подписки и ограниченные записи безопасности. Пароли хранятся только в виде защищённых хешей и не могут быть прочитаны.',
        },
        {
          title: 'Как используются данные',
          text: 'Данные нужны для входа, расчёта прогресса и серий, отправки уведомлений, защиты аккаунтов и поддержки сервиса. Они не продаются, не используются для рекламы и маркетинговых профилей.',
        },
        {
          title: 'Сторонние сервисы',
          text: 'Провайдеры хостинга, почты и уведомлений могут обрабатывать только данные, необходимые для своих функций. Sparx не передаёт историю занятий другим пользователям или посторонним лицам.',
        },
        {
          title: 'Управление и хранение',
          text: 'Данные сохраняются, пока активен аккаунт. В настройках их можно экспортировать или безвозвратно удалить вместе с аккаунтом. Временные записи безопасности удаляются автоматически.',
        },
        {
          title: 'Безопасность',
          text: 'Используются разумные технические меры защиты, но бета-сервис не может гарантировать абсолютную безопасность и непрерывную работу. Не сохраняйте в заметках особо чувствительные сведения.',
        },
      ],
    },
    terms: {
      title: 'Условия использования',
      updated: 'Действуют с 25 июля 2026 года',
      intro: 'Sparx — некоммерческий бета-сервис для отслеживания личных учебных и практических целей.',
      sections: [
        {
          title: 'Ваш аккаунт',
          text: 'Не передавайте данные для входа другим людям и используйте доступный вам email. Вы отвечаете за действия, выполненные через ваш аккаунт.',
        },
        {
          title: 'Допустимое использование',
          text: 'Нельзя обходить защиту, получать доступ к чужим данным, создавать избыточную нагрузку или использовать сервис для незаконных действий.',
        },
        {
          title: 'Ваши материалы',
          text: 'Права на цели, заметки и историю сессий остаются у вас. Сервис получает только техническое разрешение хранить и показывать эти данные вашему аккаунту.',
        },
        {
          title: 'Доступность и копии',
          text: 'Сервис предоставляется как бета-версия без гарантии непрерывной работы. Важную информацию следует дополнительно сохранять в собственном экспорте.',
        },
        {
          title: 'Прекращение использования',
          text: 'Аккаунт и его данные можно удалить в настройках. Доступ может быть ограничен, если аккаунт используется для вреда сервису или другим пользователям.',
        },
      ],
    },
  },
}

export function LegalDocument({
  type,
  language,
  onClose,
}: {
  type: LegalDocumentType
  language: LegalLanguage
  onClose: () => void
}) {
  const copy = legalContent[language]
  const page = copy[type]

  return (
    <div className="legal-page-backdrop" role="presentation">
      <article className="legal-page" aria-labelledby="legal-page-title">
        <header className="legal-page__header">
          <div className="legal-page__brand">
            <BrandMark />
            <span>{copy.product}</span>
          </div>
          <button className="icon-button icon-button--close" type="button" aria-label={copy.close} onClick={onClose}>
            <span />
            <span />
          </button>
        </header>

        <div className="legal-page__intro">
          <p>{page.updated}</p>
          <h1 id="legal-page-title">{page.title}</h1>
          <span>{page.intro}</span>
        </div>

        <div className="legal-page__sections">
          {page.sections.map((section) => (
            <section key={section.title}>
              <h2>{section.title}</h2>
              <p>{section.text}</p>
            </section>
          ))}
        </div>

        <button className="ghost-button" type="button" onClick={onClose}>{copy.close}</button>
      </article>
    </div>
  )
}
