import { DocLink } from 'component/doclink';
import { thousandSeparate } from 'component/numberformatter';
import { Result } from 'f61ui/component/result';
import { Panel, tableClassStripedHover } from 'f61ui/component/bootstrap';
import { CommandButton, CommandIcon } from 'f61ui/component/CommandButton';
import { Info } from 'f61ui/component/info';
import { SecretReveal } from 'f61ui/component/secretreveal';
import { Timestamp } from 'f61ui/component/timestamp';
import {
	ApikeyCreate,
	ApikeyRemove,
	KekGenerateOrImport,
} from 'generated/stoserver/stoservertypes_commands';
import { getApiKeys, getKeyEncryptionKeys } from 'generated/stoserver/stoservertypes_endpoints';
import { ApiKey, DocRef, KeyEncryptionKey } from 'generated/stoserver/stoservertypes_types';
import { SettingsLayout } from 'layout/settingslayout';
import * as React from 'react';

interface UsersPageState {
	apiKeys: Result<ApiKey[]>;
	keyEncryptionKeys: Result<KeyEncryptionKey[]>;
}

export default class UsersPage extends React.Component<{}, UsersPageState> {
	state: UsersPageState = {
		apiKeys: new Result<ApiKey[]>((_) => {
			this.setState({ apiKeys: _ });
		}),
		keyEncryptionKeys: new Result<KeyEncryptionKey[]>((_) => {
			this.setState({ keyEncryptionKeys: _ });
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

				<Panel
					heading={
						<div>
							Key encryption keys &nbsp;
							<Info text="Key encryption key is used to asymmetrically encrypt the files' symmetric encryption keys." />
							&nbsp;
							<DocLink doc={DocRef.DocsSecurityEncryptionIndexMd} />
						</div>
					}>
					{this.renderKeyEncryptionKeys()}
				</Panel>

				<Panel heading="API keys">{this.renderApiKeys()}</Panel>
			</SettingsLayout>
		);
	}

	private renderApiKeys() {
		const [apiKeys, loadingOrError] = this.state.apiKeys.unwrap();

		return (
			<table className={tableClassStripedHover}>
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

	private renderKeyEncryptionKeys() {
		const [keyEncryptionKeys, loadingOrError] = this.state.keyEncryptionKeys.unwrap();

		return (
			<table className={tableClassStripedHover}>
				<thead>
					<tr>
						<th>Label</th>
						<th>Kind</th>
						<th>Age</th>
						<th>Public key fingerprint</th>
					</tr>
				</thead>
				<tbody>
					{(keyEncryptionKeys || []).map((kek) => (
						<tr key={kek.Id}>
							<td>{kek.Label}</td>
							<td>
								{kek.Kind} ({thousandSeparate(kek.Bits)} bit)
							</td>
							<td>
								<Timestamp ts={kek.Created} />
							</td>
							<td>{kek.Fingerprint}</td>
						</tr>
					))}
				</tbody>
				<tfoot>
					<tr>
						<td colSpan={99}>
							<div>{loadingOrError}</div>
							<CommandButton command={KekGenerateOrImport()} />
						</td>
					</tr>
				</tfoot>
			</table>
		);
	}

	private fetchData() {
		this.state.apiKeys.load(() => getApiKeys());
		this.state.keyEncryptionKeys.load(() => getKeyEncryptionKeys());
	}
}
