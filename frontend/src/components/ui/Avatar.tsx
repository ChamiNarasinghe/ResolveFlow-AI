export type AvatarSize = "xs" | "sm" | "md" | "lg";

interface AvatarProps {
  name: string;
  size?: AvatarSize;
  className?: string;
}

const sizes: Record<AvatarSize, string> = {
  xs: "h-6 w-6 text-xs",
  sm: "h-8 w-8 text-sm",
  md: "h-9 w-9 text-sm",
  lg: "h-11 w-11 text-base",
};

function getInitials(name: string): string {
  const parts = name.trim().split(/\s+/);
  if (parts.length === 1) return parts[0].slice(0, 2).toUpperCase();
  return (parts[0][0] + parts[parts.length - 1][0]).toUpperCase();
}

// Deterministic colour from name so the same user always gets the same colour.
// Uses palette-aligned warm tones — no indigo/blue.
const colours = [
  "bg-[#838236]",   /* Olive          */
  "bg-[#6b6b2b]",   /* Olive dark     */
  "bg-[#5a7a4a]",   /* warm green     */
  "bg-[#7a5a3a]",   /* warm brown     */
  "bg-[#8a6a2a]",   /* warm amber     */
  "bg-[#6a4a5a]",   /* warm plum      */
  "bg-[#4a6a6a]",   /* warm teal      */
];

function getColour(name: string): string {
  let hash = 0;
  for (let i = 0; i < name.length; i++) {
    hash = name.charCodeAt(i) + ((hash << 5) - hash);
  }
  return colours[Math.abs(hash) % colours.length];
}

export function Avatar({ name, size = "md", className = "" }: AvatarProps) {
  return (
    <span
      className={`inline-flex shrink-0 items-center justify-center rounded-full font-semibold text-white ${getColour(name)} ${sizes[size]} ${className}`}
      aria-label={name}
      title={name}
    >
      {getInitials(name)}
    </span>
  );
}
