let colorOptions = [];

fetch('/assets/colors.json')
	.then(r => r.json())
	.then(data => {
		colorOptions = Array.isArray(data?.colors) ? data.colors : data;
		document.querySelectorAll('.row').forEach(initRow);
	});

document.getElementById('add-row').addEventListener('click', () => {
	const container = document.getElementById('rows-container');
	const row = container.querySelector('.row').cloneNode(true);
	row.querySelector('.piece-id-field').value = '';
	row.querySelector('.color-input').value = '';
	row.querySelector('.suggestions').innerHTML = '';
	row.querySelector('.suggestions').classList.remove('open');
	container.appendChild(row);
	initRow(row);
});

document.getElementById('piece-form').addEventListener('submit', e => {
	for (const input of e.target.querySelectorAll('.color-input')) {
		if (!colorOptions.some(o => o.name === input.value)) {
			input.setCustomValidity('Choose a valid color from suggestions');
			input.reportValidity();
			e.preventDefault();
			return;
		}
	}
});

function matchingColors(value) {
	const q = value.trim().toLowerCase();
	if (!q) return colorOptions;

	return colorOptions
		.filter(o => o.name.toLowerCase().includes(q))
		.sort((a, b) => {
			const [an, bn] = [a.name.toLowerCase(), b.name.toLowerCase()];
			if (an === q) return -1;
			if (bn === q) return 1;
			if (an.startsWith(q) !== bn.startsWith(q)) return an.startsWith(q) ? -1 : 1;
			return an.localeCompare(bn);
		});
}

function initRow(row) {
	const input = row.querySelector('.color-input');
	const box = row.querySelector('.suggestions');
	let activeIndex = -1;

	row.querySelector('.remove-row').addEventListener('click', () => {
		const container = document.getElementById('rows-container');
		if (container.children.length > 1) row.remove();
	});

	function validate() {
		const valid = !input.value.trim() || colorOptions.some(o => o.name === input.value.trim());
		input.setCustomValidity(valid ? '' : 'Choose a valid color from suggestions');
		return valid;
	}

	function highlight(index) {
		activeIndex = index;
		box.querySelectorAll('.suggestion-item:not(.empty)').forEach((el, i) => {
			el.classList.toggle('active', i === index);
			if (i === index) el.scrollIntoView({ block: 'nearest' });
		});
	}

	function render() {
		const matches = matchingColors(input.value);
		box.innerHTML = '';
		activeIndex = -1;

		if (!matches.length) {
			const el = document.createElement('button');
			el.type = 'button';
			el.className = 'suggestion-item empty';
			el.textContent = 'No matches';
			el.disabled = true;
			box.appendChild(el);
		}

		matches.forEach((option, i) => {
			const item = document.createElement('button');
			item.type = 'button';
			item.className = 'suggestion-item';

			const preview = document.createElement('div');
			preview.className = 'color-preview';
			preview.style.backgroundColor = '#' + option.rgb;

			const text = document.createElement('span');
			text.textContent = option.name;

			item.append(preview, text);
			item.addEventListener('mousedown', e => {
				e.preventDefault();
				input.value = option.name;
				validate();
				box.classList.remove('open');
			});
			item.addEventListener('mouseover', () => highlight(i));
			box.appendChild(item);
		});

		box.classList.add('open');
	}

	input.addEventListener('focus', render);
	input.addEventListener('input', () => { validate(); render(); });
	input.addEventListener('blur', () => {
		setTimeout(() => {
			box.classList.remove('open');
			if (input.value.trim() && !validate()) input.reportValidity();
		}, 150);
	});
	input.addEventListener('keydown', e => {
		const matches = matchingColors(input.value);
		const items = box.querySelectorAll('.suggestion-item:not(.empty)');

		if (e.key === 'ArrowDown') {
			e.preventDefault();
			if (!box.classList.contains('open')) render();
			highlight(Math.min(activeIndex + 1, items.length - 1));
		} else if (e.key === 'ArrowUp') {
			e.preventDefault();
			if (activeIndex > 0) highlight(activeIndex - 1);
		} else if (e.key === 'Enter') {
			e.preventDefault();
			const selected = activeIndex >= 0 ? matches[activeIndex] : matches[0];
			if (!selected) { if (input.value.trim()) input.reportValidity(); return; }
			input.value = selected.name;
			validate();
			box.classList.remove('open');
		}
	});
}
