const languageListItems = document.getElementsByClassName("language-item");
if (languageListItems !== undefined) {
	for (var i = 0; i < languageListItems.length; i++) {
		let languageListItem = languageListItems[i];
		languageListItem.onclick = function() {
			document.cookie = "accept-language=" + languageListItem.getAttribute("tag") + "; Path=/; Max-Age=31536000";
			location.reload();
		};
	}
}

const pres = document.getElementsByTagName("pre");
if (pres !== undefined) {
	for (var i = 0; i < pres.length; i++) {
		code = pres[i].getElementsByTagName("code")[0]
		code.setAttribute("class", code.getAttribute("class").replace("language-", ""));
		hljs.highlightBlock(code);
	}
}
