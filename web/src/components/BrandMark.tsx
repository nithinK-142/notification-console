interface BrandMarkProps {
  size?: number;
}

export function BrandMark({ size = 42 }: BrandMarkProps) {
  return <img src="/brand-mark.svg" width={size} height={size} alt="" aria-hidden="true" className="brand-mark shrink-0" />;
}
