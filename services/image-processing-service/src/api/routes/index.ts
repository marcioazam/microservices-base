import { FastifyInstance } from 'fastify';
import { uploadController } from '@api/controllers/upload.controller';
import { imageController } from '@api/controllers/image.controller';
import { jobController } from '@api/controllers/job.controller';
import { batchController } from '@api/controllers/batch.controller';

export async function registerRoutes(server: FastifyInstance): Promise<void> {
  // Upload routes
  server.post('/upload', uploadController.uploadFile.bind(uploadController));
  server.post('/upload/url', uploadController.uploadFromUrl.bind(uploadController));
  server.get('/image/:id', uploadController.getImage.bind(uploadController));

  // Image processing routes
  server.post('/image/resize', imageController.resize.bind(imageController));
  server.post('/image/convert', imageController.convert.bind(imageController));
  server.post('/image/adjust', imageController.adjust.bind(imageController));
  server.post('/image/rotate', imageController.rotate.bind(imageController));
  server.post('/image/flip', imageController.flip.bind(imageController));
  server.post('/image/watermark', imageController.watermark.bind(imageController));
  server.post('/image/compress', imageController.compress.bind(imageController));

  // Batch processing routes
  server.post('/batch/resize', batchController.batchResize.bind(batchController));

  // Job routes
  server.get('/job/:jobId', jobController.getStatus.bind(jobController));
  server.get('/job/:jobId/result', jobController.getResult.bind(jobController));
  server.delete('/job/:jobId', jobController.cancel.bind(jobController));
}
