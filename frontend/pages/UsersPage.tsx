import { Result } from 'component/result';
import { Panel } from 'f61ui/component/bootstrap';
import { CommandButton, CommandIcon } from 'f61ui/component/CommandButton';
import { SecretReveal } from 'f61ui/component/secretreveal';
import { Timestamp } from 'f61ui/component/timestamp';
import { ApikeyCreate, ApikeyRemove } from 'generated/stoserver/stoservertypes_commands';
import { getApiKeys } from 'generated/stoserver/stoservertypes_endpoints';
import { ApiKey } from 'generated/stoserver/stoservertypes_types';
import { SettingsLayout } from 'layout/settingslayout';
import * as React from 'react';

interface UsersPageState {
	apiKeys: Result<ApiKey[]>;
}

export default class UsersPage extends React.Component<{}, UsersPageState> {
	state: UsersPageState = {
		apiKeys: new Result<ApiKey[]>((_) => {
			this.setState({ apiKeys: _ });
		}),
	};

	componentDidMount() {
		this.fetchData();
	}

	componentWillReceiveProps() {
		this.fetchData();
	}

	render() {
		return (
			<SettingsLayout title="Users" breadcrumbs={[]}>
				<Panel heading="Users">
					<p>todo</p>
				</Panel>

				<Panel heading="Encryption keys">
					<p>todo</p>
				</Panel>

				<Panel heading="API keys">{this.renderApiKeys()}</Panel>
			</SettingsLayout>
		);
	}

	private renderApiKeys() {
		const [apiKeys, loadingOrError] = this.state.apiKeys.unwrap();

		return (
			<table className="table table-striped table-hover">
				<thead>
					<tr>
						<th>Name</th>
						<th>Age</th>
						<th>AuthToken</th>
						<th />
					</tr>
				</thead>
				<tbody>
					{(apiKeys || []).map((apiKey) => (
						<tr key={apiKey.Id}>
							<td>{apiKey.Name}</td>
							<td>
								<Timestamp ts={apiKey.Created} />
							</td>
							<td>
								<SecretReveal secret={apiKey.AuthToken} />
							</td>
							<td>
								<CommandIcon
									command={ApikeyRemove(apiKey.Id, {
										disambiguation: apiKey.Name,
									})}
								/>
							</td>
						</tr>
					))}
				</tbody>
				<tfoot>
					<tr>
						<td colSpan={99}>
							<div>{loadingOrError}</div>
							<CommandButton command={ApikeyCreate()} />
						</td>
					</tr>
				</tfoot>
			</table>
		);
	}

	private fetchData() {
		this.state.apiKeys.load(() => getApiKeys());
	}
}
