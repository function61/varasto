import { DocLink } from 'component/doclink';
import { RefreshButton } from 'component/refreshbutton';
import { Result } from 'component/result';
import { InfoAlert } from 'f61ui/component/alerts';
import { Panel } from 'f61ui/component/bootstrap';
import { bytesToHumanReadable } from 'f61ui/component/bytesformatter';
import { CommandButton, CommandIcon } from 'f61ui/component/CommandButton';
import { MonospaceContent } from 'f61ui/component/monospacecontent';
import { SecretReveal } from 'f61ui/component/secretreveal';
import { Timestamp } from 'f61ui/component/timestamp';
import { datetimeRFC3339 } from 'f61ui/types';
import {
	DatabaseBackup,
	DatabaseBackupConfigure,
} from 'generated/stoserver/stoservertypes_commands';
import { getConfig, getUbackupStoredBackups } from 'generated/stoserver/stoservertypes_endpoints';
import {
	CfgMetadataLastOk,
	CfgUbackupConfig,
	ConfigValue,
	DocRef,
	UbackupStoredBackup,
} from 'generated/stoserver/stoservertypes_types';
import { SettingsLayout } from 'layout/settingslayout';
import * as React from 'react';

interface MetadataBackupPageState {
	backupConfig: Result<ConfigValue>;
	backupLastOk: Result<ConfigValue>;
	backups: Result<UbackupStoredBackup[]>;
	disclaimerAck: boolean;
}

export default class MetadataBackupPage extends React.Component<{}, MetadataBackupPageState> {
	state: MetadataBackupPageState = {
		disclaimerAck: false,
		backupConfig: new Result<ConfigValue>((_) => {
			this.setState({ backupConfig: _ });
		}),
		backupLastOk: new Result<ConfigValue>((_) => {
			this.setState({ backupLastOk: _ });
		}),
		backups: new Result<UbackupStoredBackup[]>((_) => {
			this.setState({ backups: _ });
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
			<SettingsLayout title="Backups" breadcrumbs={[]}>
				{this.renderBackup()}
				<Panel
					heading={
						<div>
							Metadata backups{' '}
							<RefreshButton
								refresh={() => {
									this.refreshBackups();
								}}
							/>
						</div>
					}>
					{this.renderStoredBackups()}
				</Panel>
			</SettingsLayout>
		);
	}

	private renderBackup() {
		const [backupConfig, loadingOrError] = this.state.backupConfig.unwrap();

		if (!backupConfig) {
			return loadingOrError;
		}

		const [
			bucket,
			bucketRegion,
			accessKeyId,
			accessKeySecret,
			encryptionPublicKey,
			alertmanagerBaseUrl,
		] = backupConfig.Value ? JSON.parse(backupConfig.Value) : ['', '', '', '', '', ''];

		return (
			<Panel
				heading={
					<div>
						Metadata backup - configuration
						<CommandIcon
							command={DatabaseBackupConfigure(
								bucket,
								bucketRegion,
								accessKeyId,
								accessKeySecret,
								encryptionPublicKey,
								alertmanagerBaseUrl,
							)}
						/>{' '}
						<DocLink doc={DocRef.DocsGuideSettingUpBackupMd} />
					</div>
				}>
				<table className="table table-striped table-hover">
					<tbody>
						<tr>
							<th>Bucket</th>
							<td>{bucket}</td>
						</tr>
						<tr>
							<th>BucketRegion</th>
							<td>{bucketRegion}</td>
						</tr>
						<tr>
							<th>AccessKeyId</th>
							<td>
								<SecretReveal secret={accessKeyId} />
							</td>
						</tr>
						<tr>
							<th>AccessKeySecret</th>
							<td>
								<SecretReveal secret={accessKeySecret} />
							</td>
						</tr>
						<tr>
							<th>EncryptionPublicKey</th>
							<td>
								<MonospaceContent>{encryptionPublicKey}</MonospaceContent>
							</td>
						</tr>
						<tr>
							<th>AlertmanagerBaseUrl</th>
							<td>
								<SecretReveal secret={alertmanagerBaseUrl} />
							</td>
						</tr>
					</tbody>
					<tfoot>
						<tr>
							<td colSpan={99}>
								<CommandButton command={DatabaseBackup()} />
							</td>
						</tr>
					</tfoot>
				</table>
			</Panel>
		);
	}

	private renderStoredBackups() {
		const [backups, loadingOrError] = this.state.backups.unwrap();

		return (
			<div>
				{this.state.backupLastOk.draw((backupLastOk) =>
					backupLastOk.Value ? (
						<p>
							Last backup was <Timestamp ts={backupLastOk.Value as datetimeRFC3339} />
							.
						</p>
					) : (
						<p>No backup was ever taken :(</p>
					),
				)}

				<table className="table table-striped table-hover">
					<thead>
						<tr>
							<th>Age</th>
							<th>Description</th>
							<th>Size</th>
						</tr>
					</thead>
					<tbody>
						{(backups || []).map((backup) => (
							<tr key={backup.ID}>
								<td>
									<Timestamp ts={backup.Timestamp} />
								</td>
								<td>{backup.Description}</td>
								<td>{bytesToHumanReadable(backup.Size)}</td>
							</tr>
						))}
					</tbody>
					<tfoot>
						<tr>
							<td colSpan={99}>
								{loadingOrError}
								{!this.state.disclaimerAck ? (
									<InfoAlert>
										Backups are not shown automatically. Click refresh to load
										them.
									</InfoAlert>
								) : null}
							</td>
						</tr>
					</tfoot>
				</table>
			</div>
		);
	}

	private fetchData() {
		this.state.backupConfig.load(() => getConfig(CfgUbackupConfig));
		this.state.backupLastOk.load(() => getConfig(CfgMetadataLastOk));
	}

	private refreshBackups() {
		this.setState({ disclaimerAck: true });

		this.state.backups.load(() => getUbackupStoredBackups());
	}
}
