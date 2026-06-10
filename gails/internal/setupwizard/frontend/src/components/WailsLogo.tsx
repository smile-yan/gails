import gailsLogoWhite from '../assets/gails-logo-white-text.svg';
import gailsLogoBlack from '../assets/gails-logo-black-text.svg';

interface WailsLogoProps {
  className?: string;
  size?: number;
  theme?: 'light' | 'dark';
}

export default function WailsLogo({ className = '', size = 240, theme = 'dark' }: WailsLogoProps) {
  // White text for dark mode, black text for light mode
  const logoSrc = theme === 'dark' ? gailsLogoWhite : gailsLogoBlack;

  return (
    <img
      src={logoSrc}
      alt="Wails"
      width={size}
      className={`object-contain ${className}`}
      style={{
        filter: 'drop-shadow(0 0 60px rgba(239, 68, 68, 0.4))',
      }}
    />
  );
}
