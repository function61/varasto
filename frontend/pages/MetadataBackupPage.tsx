import { DocLink } from 'component/doclink';
import { RefreshButton } from 'component/refreshbutton';
import { InfoAlert } from 'f61ui/component/alerts';
import { Panel } from 'f61ui/component/bootstrap';
import { bytesToHumanReadable } from 'f61ui/component/bytesformatter';
import { CommandButton, CommandIcon } from 'f61ui/component/CommandButton';
import { Loading } from 'f61ui/component/loading';
import { MonospaceContent } from 'f61ui/component/monospacecontent';
import { SecretReveal } from 'f61ui/component/secretreveal';
import { Timestamp } from 'f61ui/component/timestamp';
import { datetimeRFC3339 } from 'f61ui/types';
import { shouldAlwaysSucceed } from 'f61ui/utils';
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
	backupConfig?: ConfigValue;
	backupLastOk?: ConfigValue;
	backups?: UbackupStoredBackup[];
	disclaimerAck: boolean;
}

export default class MetadataBackupPage extends React.Component<{}, MetadataBackupPageState> {
	state: MetadataBackupPageState = {
		disclaimerAck: false,
	};

	componentDidMount() {
		shouldAlwaysSucceed(this.fetchData());
	}

	componentWillReceiveProps() {
		shouldAlwaysSucceed(this.fetchData());
	}

	render() {
		return (
			<SettingsLayout title="Backups" breadcrumbs={[]}>
				{this.renderBackup()}
				<Panel
					heading={
						<div>
							Stored backups{' '}
							<RefreshButton
								refresh={() => {
									shouldAlwaysSucceed(this.refreshBackups());
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
		const backupConfig = this.state.backupConfig;
		const backupLastOk = this.state.backupLastOk;

		if (!backupConfig || !backupLastOk) {
			return <Loading />;
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
						Metadata backup
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
							<th>Last metadata backup</th>
							<td>
								{backupLastOk.Value ? (
									<Timestamp ts={backupLastOk.Value as datetimeRFC3339} />
								) : (
									'(never)'
								)}
							</td>
						</tr>
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
		const backups = this.state.backups;

		if (!backups) {
			return <Loading />;
		}

		return (
			<table className="table table-striped table-hover">
				<thead>
					<tr>
						<th>Age</th>
						<th>Description</th>
						<th>Size</th>
					</tr>
				</thead>
				<tbody>
					{backups.map((backup) => (
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
							{!this.state.disclaimerAck ? (
								<InfoAlert>
									Backups are not shown automatically. Click refresh to load them.
								</InfoAlert>
							) : (
								''
							)}
						</td>
					</tr>
				</tfoot>
			</table>
		);
	}

	private async fetchData() {
		const [backupConfig, backupLastOk] = await Promise.all([
			getConfig(CfgUbackupConfig),
			getConfig(CfgMetadataLastOk),
		]);

		this.setState({ backupConfig, backupLastOk });
	}

	private async refreshBackups() {
		this.setState({ disclaimerAck: true });

		this.setState({ backups: await getUbackupStoredBackups() });
	}
}
