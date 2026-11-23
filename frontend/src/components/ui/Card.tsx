interface CardProps {
  children: React.ReactNode;
  className?: string;
  padding?: boolean;
}

interface CardSectionProps {
  children: React.ReactNode;
  className?: string;
}

export function Card({ children, className = "", padding = true }: CardProps) {
  return (
    <div
      className={`rounded-[var(--radius-lg)] border border-[var(--border)] bg-[var(--bg-surface)] shadow-[var(--shadow-sm)] ${padding ? "p-5" : ""} ${className}`}
    >
      {children}
    </div>
  );
}

export function CardHeader({ children, className = "" }: CardSectionProps) {
  return (
    <div
      className={`border-b border-[var(--border)] px-5 py-4 ${className}`}
    >
      {children}
    </div>
  );
}

export function CardBody({ children, className = "" }: CardSectionProps) {
  return <div className={`p-5 ${className}`}>{children}</div>;
}

export function CardFooter({ children, className = "" }: CardSectionProps) {
  return (
    <div
      className={`border-t border-[var(--border)] px-5 py-4 ${className}`}
    >
      {children}
    </div>
  );
}
