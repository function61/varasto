import { reloadCurrentPage } from 'f61ui/browserutils';
import { WarningAlert } from 'f61ui/component/alerts';
import * as React from 'react';

export class SensitivityHeadsUp extends React.Component<{}, {}> {
	render() {
		const showMaxSensitivity = getMaxSensitivityFromLocalStorage();

		if (showMaxSensitivity === 0) {
			return '';
		}

		return (
			<div className="row">
				<div className="col-md-12">
					<WarningAlert>
						Showing sensitive content. &nbsp;
						<a
							className="btn btn-warning"
							onClick={() => {
								this.dropSensitivityLevel();
							}}>
							Downgrade privileges
						</a>
					</WarningAlert>
				</div>
			</div>
		);
	}

	private dropSensitivityLevel() {
		localStorage.setItem(sensitityLevelLocalStorageKey, '0');

		// FIXME: this is not idiomatic React
		reloadCurrentPage();
	}
}

const sensitityLevelLocalStorageKey = 'max_sensitivity';

export function getMaxSensitivityFromLocalStorage(): number {
	const stored = localStorage.getItem(sensitityLevelLocalStorageKey);
	if (stored !== null) {
		return +stored;
	}

	return 0;
}
