import { defineCollection, z } from 'astro:content';
import { glob } from 'astro/loaders';

const guides = defineCollection({
  loader: glob({ pattern: '**/*.md', base: './src/content/guides' }),
  schema: z.object({
    title: z.string(),
    description: z.string(),
    h1: z.string(),
    updated: z.coerce.date(),
    howto: z.boolean().default(false),
    steps: z
      .array(
        z.object({
          name: z.string(),
          text: z.string(),
        }),
      )
      .optional(),
    faq: z
      .array(
        z.object({
          q: z.string(),
          a: z.string(),
        }),
      )
      .optional(),
  }),
});

export const collections = { guides };
