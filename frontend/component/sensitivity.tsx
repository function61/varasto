import { reloadCurrentPage } from 'f61ui/browserutils';
import { Alert } from 'f61ui/component/alerts';
import { unrecognizedValue } from 'f61ui/utils';
import * as React from 'react';

export enum Sensitivity {
	FamilyFriendly = 0,
	Sensitive = 1,
	MyEyesOnly = 2,
}

export function sensitivityLabel(s: Sensitivity): string {
	switch (s) {
		case Sensitivity.FamilyFriendly:
			return 'Family friendly';
		case Sensitivity.Sensitive:
			return 'Sensitive';
		case Sensitivity.MyEyesOnly:
			return 'My eyes only';
		default:
			throw unrecognizedValue(s);
	}
}

function bootstrapDangerLevelFor(s: Sensitivity): 'info' | 'warning' | 'danger' {
	switch (s) {
		case Sensitivity.FamilyFriendly:
			return 'info'; // not actually used by SensitivityHeadsUp
		case Sensitivity.Sensitive:
			return 'warning';
		case Sensitivity.MyEyesOnly:
			return 'danger';
		default:
			throw unrecognizedValue(s);
	}
}

export class SensitivityHeadsUp extends React.Component<{}, {}> {
	render() {
		const showMaxSensitivity = getMaxSensitivityFromLocalStorage();

		if (showMaxSensitivity === Sensitivity.FamilyFriendly) {
			return '';
		}

		// ends up as btn-warning | btn-danger | ...
		const bootstrapClass = bootstrapDangerLevelFor(showMaxSensitivity);

		return (
			<div className="row">
				<div className="col-md-12">
					<Alert level={bootstrapClass}>
						Showing content: {sensitivityLabel(showMaxSensitivity)} &nbsp;
						<a
							className={`btn btn-${bootstrapClass}`}
							onClick={() => {
								this.dropSensitivityLevel();
							}}>
							Downgrade privileges
						</a>
					</Alert>
				</div>
			</div>
		);
	}

	private dropSensitivityLevel() {
		changeSensitivity(Sensitivity.FamilyFriendly);

		// FIXME: this is not idiomatic React
		reloadCurrentPage();
	}
}

const sensitityLevelLocalStorageKey = 'max_sensitivity';

export function createSensitivityAuthorizer(): (sens: Sensitivity) => boolean {
	const currSens = getMaxSensitivityFromLocalStorage();

	return (sens: Sensitivity) => currSens >= sens;
}

export function changeSensitivity(sensitivity: Sensitivity) {
	localStorage.setItem(sensitityLevelLocalStorageKey, sensitivity.toString());
}

export function getMaxSensitivityFromLocalStorage(): Sensitivity {
	const stored = localStorage.getItem(sensitityLevelLocalStorageKey);
	if (stored !== null) {
		return +stored;
	}

	return Sensitivity.FamilyFriendly;
}
