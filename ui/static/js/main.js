var navLinks = document.querySelectorAll("nav a");
for (var i = 0; i < navLinks.length; i++) {
	var link = navLinks[i]
	if (link.getAttribute('href') == window.location.pathname) {
		link.classList.add("live");
		break;
	}
}

// function increaseAmount(value) {
// 	let input = document.getElementById("amount");
// 	input.value = parseInt(input.value) + value;
// }
