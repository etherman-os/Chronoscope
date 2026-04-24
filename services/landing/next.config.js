/** @type {import('next').NextConfig} */
const nextConfig = {
  output: 'export',
  distDir: 'dist',
  images: {
    unoptimized: true,
  },
  // TODO: Add custom domain when purchased
  // assetPrefix: 'https://chronoscope.dev',
};

module.exports = nextConfig;
