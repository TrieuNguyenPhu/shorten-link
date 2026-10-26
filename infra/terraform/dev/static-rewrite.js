function handler(event) {
  var request = event.request;
  var uri = request.uri;
  if (
    uri.indexOf("/api/") === 0 ||
    uri.indexOf("/r/") === 0 ||
    uri === "/health"
  ) {
    return request;
  }
  if (uri.charAt(uri.length - 1) === "/") {
    request.uri += "index.html";
  } else if (uri.substring(uri.lastIndexOf("/") + 1).indexOf(".") === -1) {
    request.uri += "/index.html";
  }
  return request;
}
