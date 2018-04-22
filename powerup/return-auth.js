var token = window.location.hash.substring(7)
let authorize

try {
  if (window.opener && typeof window.opener.authorize === 'function') {
    authorize = window.opener.authorize
  }
} catch (e) {
  // security settings are preventing this, fallback to local storage.
}

if (authorize) {
  authorize(token)
}

setTimeout(function () { window.close() }, 1000)
