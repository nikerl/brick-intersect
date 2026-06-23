const form = document.getElementById('piece-form');
const rowsContainer = document.getElementById('rows-container');
const addRowBtn = document.getElementById('add-row');

let colorOptions = [];
let formInitialized = false;

// Fetch color options directly from static JSON
fetch('/assets/colors.json')
	.then(response => {
		console.log('Response status:', response.status);
		if (!response.ok) {
			throw new Error(`HTTP error! status: ${response.status}`);
		}
		return response.json();
	})
	.then(data => {
		console.log('Loaded colors:', data);
		colorOptions = Array.isArray(data?.colors) ? data.colors : data;
		initializeForm();
	})
	.catch(error => {
		console.error('Error loading colors:', error);
		console.error('Error details:', error.message);
		colorOptions = [
			{ name: 'Option A', rgb: 'FF0000' },
			{ name: 'Test', rgb: '00FF00' },
			{ name: 'sosserier', rgb: '0000FF' }
		]; // Fallback
		initializeForm();
	});

function initializeForm() {
	if (formInitialized) return;
	formInitialized = true;

	// Initialize existing rows
	document.querySelectorAll('.row').forEach(initRow);

	// Add new row button
	addRowBtn.addEventListener('click', () => {
		const defaultRow = rowsContainer.querySelector('.row');
		const newRow = defaultRow.cloneNode(true);
		newRow.querySelector('.piece-id-field').value = '';
		newRow.querySelector('.color-input').value = '';
		newRow.querySelector('.suggestions').innerHTML = '';
		newRow.querySelector('.suggestions').classList.remove('open');
		rowsContainer.appendChild(newRow);
		initRow(newRow);
	});

	// Form submit handler
	form.addEventListener('submit', (event) => {
		const colorInputs = form.querySelectorAll('.color-input');
		let isValid = true;
		colorInputs.forEach(input => {
			if (!colorOptions.some(option => option.name === input.value)) {
				input.setCustomValidity('Choose a valid color format from suggestions');
				input.reportValidity();
				isValid = false;
			}
		});
		if (!isValid) {
			event.preventDefault();
		}
	});
}


function getMatchingOptions(value) {
	const normalizedValue = value.trim().toLowerCase();
	if (!normalizedValue) return colorOptions;

	// Sort matches by relevance: exact match > starts with > contains
	const matches = colorOptions.filter((option) => {
		const optionName = option.name.toLowerCase();
		return optionName.includes(normalizedValue);
	});

	matches.sort((a, b) => {
		const aName = a.name.toLowerCase();
		const bName = b.name.toLowerCase();

		// Exact match
		if (aName === normalizedValue) return -1;
		if (bName === normalizedValue) return 1;

		// Starts with
		if (aName.startsWith(normalizedValue) && !bName.startsWith(normalizedValue)) return -1;
		if (bName.startsWith(normalizedValue) && !aName.startsWith(normalizedValue)) return 1;

		// Both start with or both don't, sort alphabetically
		return aName.localeCompare(bName);
	});

	return matches;
}

function initRow(row) {
	const colorInput = row.querySelector('.color-input');
	const suggestionBox = row.querySelector('.suggestions');
	const removeBtn = row.querySelector('.remove-row');
	let activeIndex = -1;

	removeBtn.addEventListener('click', () => {
		if (rowsContainer.children.length > 1) {
			row.remove();
		}
	});

	function validateInput() {
		const value = colorInput.value.trim();

		// Allow empty values while the user is navigating fields; required validation
		// will still run on submit.
		if (!value) {
			colorInput.setCustomValidity('');
			return true;
		}

		const isValid = colorOptions.some(option => option.name === value);
		colorInput.setCustomValidity(isValid ? '' : 'Choose a valid color from suggestions');
		return isValid;
	}

	function reportInvalidInput() {
		const value = colorInput.value.trim();
		const isValid = validateInput();

		// Do not force focus back when the field is empty.
		if (value && !isValid) {
			colorInput.reportValidity();
		}
	}

	function closeSuggestions() {
		suggestionBox.classList.remove('open');
	}

	function renderSuggestions() {
		const matches = getMatchingOptions(colorInput.value);
		suggestionBox.innerHTML = '';
		activeIndex = -1;

		if (!matches.length) {
			const emptyState = document.createElement('button');
			emptyState.type = 'button';
			emptyState.className = 'suggestion-item empty';
			emptyState.textContent = 'No matches';
			emptyState.disabled = true;
			suggestionBox.appendChild(emptyState);
		}

		matches.forEach((option, index) => {
			const item = document.createElement('button');
			item.type = 'button';
			item.className = 'suggestion-item';
			item.dataset.index = index;

			// Create color preview
			const preview = document.createElement('div');
			preview.className = 'color-preview';
			preview.style.backgroundColor = '#' + option.rgb;

			// Create text span
			const text = document.createElement('span');
			text.textContent = option.name;

			item.appendChild(preview);
			item.appendChild(text);

			item.addEventListener('mousedown', (event) => {
				event.preventDefault();
				colorInput.value = option.name;
				validateInput();
				closeSuggestions();
			});

			item.addEventListener('mouseover', () => {
				setActiveIndex(index);
			});

			suggestionBox.appendChild(item);
		});

		suggestionBox.classList.add('open');
	}

	function setActiveIndex(index) {
		activeIndex = index;
		const items = suggestionBox.querySelectorAll('.suggestion-item:not(.empty)');
		items.forEach((item, i) => {
			if (i === index) {
				item.classList.add('active');
			} else {
				item.classList.remove('active');
			}
		});
	}

	colorInput.addEventListener('focus', renderSuggestions);
	colorInput.addEventListener('input', () => {
		validateInput();
		renderSuggestions();
	});
	colorInput.addEventListener('blur', () => {
		window.setTimeout(() => {
			closeSuggestions();
			reportInvalidInput();
		}, 150);
	});
	colorInput.addEventListener('keydown', (event) => {
		const matches = getMatchingOptions(colorInput.value);
		const items = suggestionBox.querySelectorAll('.suggestion-item:not(.empty)');

		if (event.key === 'ArrowDown') {
			event.preventDefault();
			if (!suggestionBox.classList.contains('open')) {
				renderSuggestions();
			}
			setActiveIndex(Math.min(activeIndex + 1, items.length - 1));
		} else if (event.key === 'ArrowUp') {
			event.preventDefault();
			if (activeIndex > 0) {
				setActiveIndex(activeIndex - 1);
			}
		} else if (event.key === 'Enter') {
			event.preventDefault();

			let selectedOption;
			if (activeIndex >= 0 && activeIndex < matches.length) {
				selectedOption = matches[activeIndex];
			} else {
				selectedOption = matches[0];
			}

			if (!selectedOption) {
				reportInvalidInput();
				return;
			}

			colorInput.value = selectedOption.name;
			validateInput();
			closeSuggestions();
		}
	});
}

