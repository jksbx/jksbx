// Web driver.
await page.evaluateOnNewDocument(() => {
  Object.defineProperty(navigator, 'webdriver', {
    get: () => false,
  });
});

// Chrome
await page.evaluateOnNewDocument(() => {
  // We can mock this in as much depth as we need for the test.
  window.navigator.chrome = {
    runtime: {},
    // etc.
  };
});

// Permissions
await page.evaluateOnNewDocument(() => {
  const originalQuery = window.navigator.permissions.query;
  return window.navigator.permissions.query = (parameters) => (
    parameters.name === 'notifications' ?
      Promise.resolve({ state: Notification.permission }) :
      originalQuery(parameters)
  );
});

// Plugin length.
await page.evaluateOnNewDocument(() => {
  // Overwrite the `plugins` property to use a custom getter.
  Object.defineProperty(navigator, 'plugins', {
    // This just needs to have `length > 0` for the current test,
    // but we could mock the plugins too if necessary.
    get: () => [1, 2, 3, 4, 5, 6],
  });
});

// Languages
await page.evaluateOnNewDocument(() => {
  // Overwrite the `plugins` property to use a custom getter.
  Object.defineProperty(navigator, 'languages', {
    get: () => ['en-US', 'en'],
  });
});

// WebGL vendor and renderer
const getParameter = WebGLRenderingContext.getParameter;
WebGLRenderingContext.prototype.getParameter = function(parameter) {
  // UNMASKED_VENDOR_WEBGL
  if (parameter === 37445) {
    return 'Intel Open Source Technology Center';
  }
  // UNMASKED_RENDERER_WEBGL
  if (parameter === 37446) {
    return 'Mesa DRI Intel(R) Ivybridge Mobile ';
  }

  return getParameter(parameter);
};

// Broken image
['height', 'width'].forEach(property => {
  // store the existing descriptor
  const imageDescriptor = Object.getOwnPropertyDescriptor(HTMLImageElement.prototype, property);

  // redefine the property with a patched descriptor
  Object.defineProperty(HTMLImageElement.prototype, property, {
    ...imageDescriptor,
    get: function() {
      // return an arbitrary non-zero dimension if the image failed to load
      if (this.complete && this.naturalHeight == 0) {
        return 20;
      }
      // otherwise, return the actual dimension
      return imageDescriptor.get.apply(this);
    },
  });
});

// Hairline feature
const elementDescriptor = Object.getOwnPropertyDescriptor(HTMLElement.prototype, 'offsetHeight');
// redefine the property with a patched descriptor
Object.defineProperty(HTMLDivElement.prototype, 'offsetHeight', {
  ...elementDescriptor,
  get: function() {
    if (this.id === 'modernizr') {
        return 1;
    }
    return elementDescriptor.get.apply(this);
  },
});
