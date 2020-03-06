const pres = document.getElementsByTagName("pre");
if (pres !== undefined) {
	for (var i = 0; i < pres.length; i++) {
		code = pres[i].getElementsByTagName("code")[0]
		code.setAttribute("class", code.getAttribute("class").replace("language-", ""));
		hljs.highlightBlock(code);
	}
}
