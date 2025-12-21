import { getReleases } from '@/lib/github';
import styles from './releases.module.css';

export const metadata = {
  title: 'Releases - acc',
  description: 'All acc releases and changelogs',
};

export default async function ReleasesPage() {
  const releases = await getReleases(10);

  if (releases.length === 0) {
    return (
      <div className="container">
        <h1>Releases</h1>
        <p>Unable to fetch releases. Please visit the <a href="https://github.com/cloudcwfranck/acc/releases">GitHub Releases page</a>.</p>
      </div>
    );
  }

  return (
    <div className="container">
      <div className={styles.header}>
        <h1>Releases</h1>
        <p>All acc releases and changelogs</p>
      </div>

      <div className={styles.releases}>
        {releases.map((release) => {
          const publishedDate = new Date(release.published_at).toLocaleDateString('en-US', {
            year: 'numeric',
            month: 'long',
            day: 'numeric',
          });

          // Extract first paragraph or first 300 chars of body
          const bodyPreview = release.body
            ? release.body.split('\n\n')[0].substring(0, 300) +
              (release.body.length > 300 ? '...' : '')
            : 'No release notes available.';

          return (
            <div key={release.id} className={styles.release}>
              <div className={styles.releaseHeader}>
                <div>
                  <h2 className={styles.releaseName}>
                    <a
                      href={release.html_url}
                      target="_blank"
                      rel="noopener noreferrer"
                    >
                      {release.name || release.tag_name}
                    </a>
                  </h2>
                  <div className={styles.releaseMeta}>
                    <span className={styles.tag}>{release.tag_name}</span>
                    <span className={styles.date}>{publishedDate}</span>
                  </div>
                </div>
              </div>

              <div className={styles.releaseBody}>
                <p>{bodyPreview}</p>
              </div>

              <div className={styles.releaseFooter}>
                <a
                  href={release.html_url}
                  target="_blank"
                  rel="noopener noreferrer"
                  className={styles.viewLink}
                >
                  View full release notes →
                </a>
                {release.assets.length > 0 && (
                  <span className={styles.assetsCount}>
                    {release.assets.length} assets
                  </span>
                )}
              </div>
            </div>
          );
        })}
      </div>

      <div className={styles.moreReleases}>
        <p>
          View all releases on{' '}
          <a
            href="https://github.com/cloudcwfranck/acc/releases"
            target="_blank"
            rel="noopener noreferrer"
          >
            GitHub
          </a>
        </p>
      </div>
    </div>
  );
}
