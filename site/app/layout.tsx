import type { Metadata } from 'next';
import './globals.css';
import Navigation from '@/components/Navigation';
import Footer from '@/components/Footer';
import PrereleaseBannerWrapper from '@/components/PrereleaseBannerWrapper';

export const metadata: Metadata = {
  title: 'acc - Policy Verification CLI',
  description: 'acc is a policy verification CLI that turns cloud controls into deterministic, explainable results for CI/CD.',
  keywords: ['policy', 'verification', 'CLI', 'CI/CD', 'security', 'compliance'],
  authors: [{ name: 'acc team' }],
  openGraph: {
    title: 'acc - Policy Verification CLI',
    description: 'Policy verification CLI for deterministic, explainable results',
    type: 'website',
  },
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en">
      <body>
        <PrereleaseBannerWrapper />
        <Navigation />
        <main>{children}</main>
        <Footer />
      </body>
    </html>
  );
}
