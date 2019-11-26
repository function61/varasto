import { Result } from 'component/result';
import { Panel, Well } from 'f61ui/component/bootstrap';
import { CommandButton, CommandIcon } from 'f61ui/component/CommandButton';
import {
	ConfigSetFuseServerBaseurl,
	ConfigSetNetworkShareBaseUrl,
	FuseUnmountAll,
} from 'generated/stoserver/stoservertypes_commands';
import { getConfig } from 'generated/stoserver/stoservertypes_endpoints';
import {
	CfgFuseServerBaseUrl,
	CfgNetworkShareBaseUrl,
	ConfigValue,
} from 'generated/stoserver/stoservertypes_types';
import { SettingsLayout } from 'layout/settingslayout';
import * as React from 'react';

interface FuseServerPageState {
	baseUrl: Result<ConfigValue>;
	networkShareBaseUrl: Result<ConfigValue>;
}

export default class FuseServerPage extends React.Component<{}, FuseServerPageState> {
	state: FuseServerPageState = {
		baseUrl: new Result<ConfigValue>((_) => {
			this.setState({ baseUrl: _ });
		}),
		networkShareBaseUrl: new Result<ConfigValue>((_) => {
			this.setState({ networkShareBaseUrl: _ });
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
			<SettingsLayout title="FUSE server &amp; network folders" breadcrumbs={[]}>
				<Panel heading="Settings">{this.renderEditForm()}</Panel>
			</SettingsLayout>
		);
	}

	private renderEditForm() {
		return (
			<div className="form-horizontal">
				{this.state.baseUrl.draw((baseUrl) => (
					<div className="form-group">
						<label className="col-sm-2 control-label">
							FUSE server base URL
							<CommandIcon command={ConfigSetFuseServerBaseurl(baseUrl.Value)} />
						</label>
						<div className="col-sm-10">
							{baseUrl.Value !== ''
								? baseUrl.Value
								: 'Not set - unable to mount network folders'}
						</div>
					</div>
				))}

				{this.state.networkShareBaseUrl.draw((networkShareBaseUrl) => (
					<div className="form-group">
						<label className="col-sm-2 control-label">
							Network share base URL
							<CommandIcon
								command={ConfigSetNetworkShareBaseUrl(networkShareBaseUrl.Value)}
							/>
						</label>
						<div className="col-sm-10">
							{networkShareBaseUrl.Value !== ''
								? networkShareBaseUrl.Value
								: 'Not set'}
						</div>
					</div>
				))}

				<CommandButton command={FuseUnmountAll()} />

				<Well>
					<p>FUSE is technology in Linux where we can easily define filesystems.</p>

					<p>
						Varasto supports projecting Varasto collections over FUSE as read-only
						files. Great use cases are direct-streaming videos. If you want transcoded
						videos, those you can view directly from Varasto's web ui.
					</p>

					<p>
						You need to run this Varasto-FUSE process separately from the main Varasto
						binary.
					</p>

					<p>
						You can then export the filesystem over Samba as a read-only network folder
						to other computers in the network.
					</p>
				</Well>
			</div>
		);
	}

	private fetchData() {
		this.state.baseUrl.load(() => getConfig(CfgFuseServerBaseUrl));
		this.state.networkShareBaseUrl.load(() => getConfig(CfgNetworkShareBaseUrl));
	}
}
