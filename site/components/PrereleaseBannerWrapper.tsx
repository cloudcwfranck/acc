import { getLatestStableRelease, getLatestPrerelease, isPrereleaseNewer } from '@/lib/github';
import PrereleaseBanner from './PrereleaseBanner';

export default async function PrereleaseBannerWrapper() {
  const [stable, prerelease] = await Promise.all([
    getLatestStableRelease(),
    getLatestPrerelease(),
  ]);

  // Only show banner if prerelease exists and is newer than stable
  if (!prerelease) {
    return null;
  }

  if (stable && !isPrereleaseNewer(prerelease, stable)) {
    return null;
  }

  return <PrereleaseBanner version={prerelease.tag_name} />;
}
