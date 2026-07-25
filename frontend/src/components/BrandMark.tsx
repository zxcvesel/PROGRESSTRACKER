type BrandMarkProps = {
  className?: string
}

export function BrandMark({ className = '' }: BrandMarkProps) {
  return (
    <svg
      className={`brand-mark ${className}`.trim()}
      viewBox="0 0 64 64"
      aria-hidden="true"
    >
      <path
        className="brand-mark__flame"
        d="M33.4 17.2c1.1 5.8-1.5 9.5-4.8 12.8-2.6 2.7-4.9 5.2-4.9 9.1 0 5.2 4 8.9 9.1 8.9 5.2 0 9.2-3.7 9.2-8.9 0-3.6-1.8-6.7-4.9-9.7.2 2.9-.9 5.1-3.2 6.5-.9-5.4 1.6-9.7-.5-18.7Z"
      />
    </svg>
  )
}
