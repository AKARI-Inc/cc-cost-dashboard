function doGet(e) {
  return HtmlService.createHtmlOutputFromFile('client/index')
    .setTitle('Claude Code Cost Dashboard')
    .setXFrameOptionsMode(HtmlService.XFrameOptionsMode.ALLOWALL);
}
