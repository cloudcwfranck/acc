import { NextRequest, NextResponse } from 'next/server';

const GITHUB_API_BASE = 'https://api.github.com';
const REPO_OWNER = 'cloudcwfranck';
const REPO_NAME = 'acc';

export async function GET(request: NextRequest) {
  const searchParams = request.nextUrl.searchParams;
  const limit = searchParams.get('limit') || '10';

  const headers: Record<string, string> = {
    'Accept': 'application/vnd.github.v3+json',
  };

  // Use GITHUB_TOKEN if available (server-side only)
  if (process.env.GITHUB_TOKEN) {
    headers['Authorization'] = `Bearer ${process.env.GITHUB_TOKEN}`;
  }

  try {
    const response = await fetch(
      `${GITHUB_API_BASE}/repos/${REPO_OWNER}/${REPO_NAME}/releases?per_page=${limit}`,
      {
        headers,
        next: { revalidate: 60 }, // ISR with 60s revalidation
      }
    );

    if (!response.ok) {
      return NextResponse.json(
        { error: `GitHub API returned ${response.status}` },
        { status: response.status }
      );
    }

    const releases = await response.json();
    return NextResponse.json(releases);
  } catch (error) {
    console.error('Error fetching releases:', error);
    return NextResponse.json(
      { error: 'Failed to fetch releases' },
      { status: 500 }
    );
  }
}
