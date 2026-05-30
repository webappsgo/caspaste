/**
 * This file is part of CasPaste.
 * CasPaste is free software released under the MIT License.
 * See LICENSE.md file for details.
 *
 * CasPaste Service Worker
 * Provides offline support and caching for PWA functionality
 */

const CACHE_NAME = 'caspaste-v1';
const STATIC_ASSETS = [
  '/',
  '/style.css',
  '/main.js',
  '/history.js',
  '/manifest.json'
];

// Install event - cache static assets
self.addEventListener('install', function(event) {
  event.waitUntil(
    caches.open(CACHE_NAME).then(function(cache) {
      return cache.addAll(STATIC_ASSETS).catch(function(error) {
        console.log('Cache addAll failed:', error);
        // Continue even if some assets fail to cache
        return Promise.resolve();
      });
    })
  );
  self.skipWaiting();
});

// Activate event - clean up old caches
self.addEventListener('activate', function(event) {
  event.waitUntil(
    caches.keys().then(function(cacheNames) {
      return Promise.all(
        cacheNames.map(function(cacheName) {
          if (cacheName !== CACHE_NAME) {
            return caches.delete(cacheName);
          }
        })
      );
    })
  );
  self.clients.claim();
});

// Fetch event - serve from cache, fallback to network
self.addEventListener('fetch', function(event) {
  // Skip non-GET requests
  if (event.request.method !== 'GET') {
    return;
  }

  // Skip chrome-extension and other non-http(s) requests
  if (!event.request.url.startsWith('http')) {
    return;
  }

  event.respondWith(
    caches.match(event.request).then(function(cachedResponse) {
      if (cachedResponse) {
        return cachedResponse;
      }

      return fetch(event.request).then(function(response) {
        // Don't cache non-successful responses
        if (!response || response.status !== 200 || response.type === 'error') {
          return response;
        }

        // Clone the response
        var responseToCache = response.clone();

        caches.open(CACHE_NAME).then(function(cache) {
          // Only cache static assets and GET requests
          if (event.request.method === 'GET' &&
            (event.request.url.endsWith('.css') ||
             event.request.url.endsWith('.js') ||
             event.request.url.endsWith('.json'))) {
            cache.put(event.request, responseToCache);
          }
        });

        return response;
      });
    })
  );
});
