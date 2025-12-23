'use client';

import { useEffect, useRef } from 'react';

interface DemoPlayerProps {
  asciinemaId?: string;
  localCastPath?: string;
}

export default function DemoPlayer({ asciinemaId, localCastPath }: DemoPlayerProps) {
  const playerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    // Only run on client side
    if (typeof window === 'undefined') return;

    // Load asciinema-player dynamically
    const loadPlayer = async () => {
      try {
        // Load CSS
        const link = document.createElement('link');
        link.rel = 'stylesheet';
        link.href = 'https://cdn.jsdelivr.net/npm/asciinema-player@3.7.0/dist/bundle/asciinema-player.min.css';
        document.head.appendChild(link);

        // Load JS
        const script = document.createElement('script');
        script.src = 'https://cdn.jsdelivr.net/npm/asciinema-player@3.7.0/dist/bundle/asciinema-player.min.js';
        script.async = true;

        script.onload = () => {
          if (playerRef.current && (window as any).AsciinemaPlayer) {
            const AsciinemaPlayer = (window as any).AsciinemaPlayer;

            // Clear any existing content
            playerRef.current.innerHTML = '';

            // Create player
            if (asciinemaId) {
              // Use asciinema.org ID
              AsciinemaPlayer.create(
                `https://asciinema.org/a/${asciinemaId}.cast`,
                playerRef.current,
                {
                  autoPlay: false,
                  loop: false,
                  speed: 1.5,
                  theme: 'asciinema',
                  fit: 'width',
                  terminalFontSize: '14px',
                }
              );
            } else if (localCastPath) {
              // Use local cast file
              AsciinemaPlayer.create(localCastPath, playerRef.current, {
                autoPlay: false,
                loop: false,
                speed: 1.5,
                theme: 'asciinema',
                fit: 'width',
                terminalFontSize: '14px',
              });
            }
          }
        };

        document.body.appendChild(script);

        return () => {
          document.head.removeChild(link);
          document.body.removeChild(script);
        };
      } catch (error) {
        console.error('Failed to load asciinema player:', error);
      }
    };

    loadPlayer();
  }, [asciinemaId, localCastPath]);

  // Fallback if no demo available
  if (!asciinemaId && !localCastPath) {
    return (
      <div style={{
        padding: '2rem',
        textAlign: 'center',
        background: 'rgba(var(--foreground-rgb), 0.03)',
        borderRadius: '8px',
        border: '1px solid rgba(var(--foreground-rgb), 0.1)'
      }}>
        <div style={{ fontSize: '2rem', marginBottom: '1rem' }}>📹</div>
        <p style={{ color: 'rgba(var(--foreground-rgb), 0.6)' }}>
          Demo recording not yet available
        </p>
      </div>
    );
  }

  return (
    <div
      ref={playerRef}
      style={{
        width: '100%',
        maxWidth: '900px',
        margin: '0 auto',
        borderRadius: '8px',
        overflow: 'hidden',
        boxShadow: '0 4px 6px rgba(0, 0, 0, 0.1)',
      }}
    />
  );
}
