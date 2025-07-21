import { DangerAlert } from 'f61ui/component/alerts';
import { Loading } from 'f61ui/component/loading';
import {
	asError,
	coerceToStructuredErrorResponse,
	formatStructuredErrorResponse,
	handleKnownGlobalErrors,
} from 'f61ui/errors';
import { shouldAlwaysSucceed } from 'f61ui/utils';
import * as React from 'react';
import * as Autocomplete from 'react-autocomplete';

interface SearchBoxProps<T> {
	dataSource: (query: string) => Promise<T[]>;
	itemToAutocompleteItem: (item: T) => AutocompleteItem;
	onSelect: (item: AutocompleteItem) => void;
	allowEmptySearch: boolean;
	searchTerm?: string; // initial
	placeholder?: string;
	autoFocus?: boolean;
}

interface SearchBoxState {
	searchTerm?: string;
	items: AutocompleteItem[];
	searchError?: string;
	loading: boolean;
}

interface AutocompleteItem {
	label: string;
	key: string;
}

export class SearchBox<T> extends React.Component<SearchBoxProps<T>, SearchBoxState> {
	state: SearchBoxState = { loading: false, items: [], searchTerm: this.props.searchTerm };
	private loading = false;
	private beginSearchTimeout?: ReturnType<typeof setTimeout>;
	private queuedQuery = '';

	render() {
		return (
			<div>
				<Autocomplete
					menuStyle={{
						// everything same as in base except for "position"
						borderRadius: '3px',
						boxShadow: '0 2px 12px rgba(0, 0, 0, 0.1)',
						background: 'rgba(255, 255, 255, 0.9)',
						padding: '2px 0',
						fontSize: '90%',
						position: 'static', // 'fixed' doesn't work inside modal
						overflow: 'auto',
						maxHeight: '50%',
					}}
					inputProps={{
						className: 'form-control',
						autoFocus: this.props.autoFocus,
						placeholder: this.props.placeholder,
						onKeyPress: (e: any) => {
							// https://github.com/reactjs/react-autocomplete/issues/338
							if (e.key !== 'Enter') {
								return;
							}

							// => user hit enter with non-suggested term => go to search results
							const searchTerm = e.target.value;

							if (searchTerm !== '') {
								alert('non-suggested term entered');
							} else {
								alert('box cleared');
							}

							e.preventDefault();
						},
					}}
					getItemValue={(item: AutocompleteItem) => item.key}
					items={this.state.items}
					renderItem={(item: AutocompleteItem, isHighlighted: boolean) => (
						<div
							key={item.key}
							style={{ background: isHighlighted ? 'lightgray' : 'white' }}>
							{item.label}
						</div>
					)}
					value={this.state.searchTerm}
					onChange={(_: React.ChangeEvent<HTMLInputElement>, text: string) => {
						this.setState({ searchTerm: text, items: [], searchError: undefined });
						this.props.onSelect({ key: '', label: '' });
						this.maybeStartSearch(text);
					}}
					onSelect={(_: string, item: AutocompleteItem) => {
						this.props.onSelect(item);
						this.setState({ searchTerm: item.label });
					}}
				/>
				{this.state.loading && <Loading />}
				{this.state.searchError && <DangerAlert>{this.state.searchError}</DangerAlert>}
			</div>
		);
	}

	private maybeStartSearch(term: string) {
		this.queuedQuery = term;

		if (this.loading) {
			return;
		}

		if (this.beginSearchTimeout) {
			clearTimeout(this.beginSearchTimeout);
			this.beginSearchTimeout = undefined;
		}

		this.beginSearchTimeout = setTimeout(() => {
			this.beginSearchTimeout = undefined;

			shouldAlwaysSucceed(this.search(this.queuedQuery));
		}, 800);
	}

	private async search(term: string) {
		this.queuedQuery = '';

		this.setLoadingState(true);

		try {
			const isEmpty = term === '';

			const searchResult =
				!isEmpty || this.props.allowEmptySearch ? await this.props.dataSource(term) : [];

			this.setState({ items: searchResult.map(this.props.itemToAutocompleteItem) });
		} catch (err) {
			const ser = coerceToStructuredErrorResponse(asError(err));
			if (handleKnownGlobalErrors(ser)) {
				return;
			}

			this.setState({ searchError: formatStructuredErrorResponse(ser) });
		}

		this.setLoadingState(false);

		// while we were fetching data from server, user wanted another query?
		if (this.queuedQuery) {
			this.maybeStartSearch(this.queuedQuery);
		}
	}

	private setLoadingState(to: boolean) {
		// React semantics don't guarantee setState({ x: ... }) to update this.state.x right away.
		// that is stupid, since that could be done immediately even with render() deferring
		this.loading = to;

		this.setState({ loading: to });
	}
}
