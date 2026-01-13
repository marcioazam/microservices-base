import sharp from 'sharp';

export async function generateTestImage(
  width = 100,
  height = 100,
  format: 'png' | 'jpeg' | 'webp' = 'png',
  hasAlpha = true
): Promise<Buffer> {
  const channels = hasAlpha ? 4 : 3;
  const pixels = Buffer.alloc(width * height * channels);

  // Create a simple gradient pattern
  for (let y = 0; y < height; y++) {
    for (let x = 0; x < width; x++) {
      const idx = (y * width + x) * channels;
      pixels[idx] = Math.floor((x / width) * 255); // R
      pixels[idx + 1] = Math.floor((y / height) * 255); // G
      pixels[idx + 2] = 128; // B
      if (hasAlpha) {
        pixels[idx + 3] = 255; // A
      }
    }
  }

  const image = sharp(pixels, {
    raw: {
      width,
      height,
      channels,
    },
  });

  switch (format) {
    case 'jpeg':
      return image.jpeg().toBuffer();
    case 'webp':
      return image.webp().toBuffer();
    default:
      return image.png().toBuffer();
  }
}

export async function generateTransparentImage(
  width = 100,
  height = 100
): Promise<Buffer> {
  const channels = 4;
  const pixels = Buffer.alloc(width * height * channels);

  // Create image with transparent regions
  for (let y = 0; y < height; y++) {
    for (let x = 0; x < width; x++) {
      const idx = (y * width + x) * channels;
      pixels[idx] = 255; // R
      pixels[idx + 1] = 0; // G
      pixels[idx + 2] = 0; // B
      // Make half the image transparent
      pixels[idx + 3] = x < width / 2 ? 0 : 255; // A
    }
  }

  return sharp(pixels, {
    raw: {
      width,
      height,
      channels,
    },
  })
    .png()
    .toBuffer();
}

export async function getImageDimensions(buffer: Buffer): Promise<{ width: number; height: number }> {
  const metadata = await sharp(buffer).metadata();
  return {
    width: metadata.width || 0,
    height: metadata.height || 0,
  };
}

export async function getImageFormat(buffer: Buffer): Promise<string> {
  const metadata = await sharp(buffer).metadata();
  return metadata.format || 'unknown';
}

export async function getPixelColor(
  buffer: Buffer,
  x: number,
  y: number
): Promise<{ r: number; g: number; b: number; a?: number }> {
  const { data, info } = await sharp(buffer)
    .raw()
    .toBuffer({ resolveWithObject: true });

  const idx = (y * info.width + x) * info.channels;
  return {
    r: data[idx],
    g: data[idx + 1],
    b: data[idx + 2],
    a: info.channels === 4 ? data[idx + 3] : undefined,
  };
}

export function buffersEqual(a: Buffer, b: Buffer): boolean {
  if (a.length !== b.length) return false;
  return a.compare(b) === 0;
}

export async function pixelsEqual(a: Buffer, b: Buffer, tolerance = 0): Promise<boolean> {
  const [dataA, dataB] = await Promise.all([
    sharp(a).raw().toBuffer({ resolveWithObject: true }),
    sharp(b).raw().toBuffer({ resolveWithObject: true }),
  ]);

  if (dataA.info.width !== dataB.info.width || dataA.info.height !== dataB.info.height) {
    return false;
  }

  for (let i = 0; i < dataA.data.length; i++) {
    if (Math.abs(dataA.data[i] - dataB.data[i]) > tolerance) {
      return false;
    }
  }

  return true;
}

export function calculateAspectRatio(width: number, height: number): number {
  return width / height;
}
