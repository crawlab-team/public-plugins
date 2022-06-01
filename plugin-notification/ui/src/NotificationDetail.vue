<template>
  <div class="detail-layout notification-detail">
    <div class="content">
      <cl-nav-tabs
          :active-key="activeKey"
          :items="tabs"
          @select="onTabSelect"
      />
      <cl-nav-actions
          class="nav-actions"
          :collapsed="false"
      >
        <cl-nav-action-group-detail-common
            @back="onBack"
            @save="onSave"
        />
      </cl-nav-actions>

      <template v-if="activeKey === 'overview'">
        <NotificationForm
            class="content-container"
            ref="formRef"
            v-model="form"
        />
      </template>
      <NotificationDetailTabTriggers
          class="content-container"
          v-else-if="activeKey === 'triggers'"
          :form="form"
          :trigger-list="triggerList"
          @change="onTriggersChange"
      />
      <NotificationDetailTabTemplate
          v-else-if="activeKey === 'template'"
          :form="form"
          @title-change="onTitleChange"
          @template-change="onTemplateChange"
      />
    </div>
  </div>
</template>

<script lang="ts">
import {computed, defineComponent, onMounted, ref} from 'vue';
import {useRoute, useRouter} from 'vue-router';
import {useRequest} from 'crawlab-ui';
import {ElMessage} from 'element-plus';
import NotificationForm from './NotificationForm.vue';
import NotificationDetailTabTemplate from './NotificationDetailTabTemplate.vue';
import NotificationDetailTabTriggers from './NotificationDetailTabTriggers.vue';

const pluginName = 'notification';
const t = (path) => window['_tp'](pluginName, path);
const _t = window['_t'];

const endpoint = '/plugin-proxy/notification';

const {
  get,
  post,
} = useRequest();

export default defineComponent({
  name: 'NotificationDetail',
  components: {
    NotificationDetailTabTriggers,
    NotificationForm,
    NotificationDetailTabTemplate,
  },
  setup() {
    const router = useRouter();

    const route = useRoute();

    const id = computed(() => route.params.id);

    const activeKey = ref('overview');

    const tabs = computed(() => [
      {
        id: 'overview',
        title: t('tabs.overview'),
      },
      {
        id: 'triggers',
        title: t('tabs.triggers'),
      },
      {
        id: 'template',
        title: t('tabs.template'),
      },
    ]);

    const form = ref({});

    const formRef = ref();

    const triggerList = ref([]);

    const onBack = () => {
      router.push(`/notifications`);
    };

    const onSave = async () => {
      if (formRef.value) await formRef.value.validate();
      await post(`${endpoint}/settings/${id.value}`, form.value);
      ElMessage.success(_t('common.message.success.save'));
    };

    const triggerActionLabelMap = {
      add: t('actions.add'),
      save: t('actions.save'),
      change: t('actions.change'),
      delete: t('actions.delete'),
    };

    const getTriggerList = async () => {
      const res = await get(`${endpoint}/triggers`);
      const {data} = res;
      triggerList.value = data.map(t => {
        const arr = t.split(':');
        const colName = arr[1];
        const model = colName.substr(0, arr[1].length - 1);
        const modelName = model
            .split('_')
            .map(w => w.split('').map((c, i) => i === 0 ? c.toUpperCase() : c).join(''))
            .join(' ');
        const action = arr[2];
        const actionLabel = triggerActionLabelMap[action];
        const label = `${modelName} ${actionLabel}`;
        return {
          key: t,
          label,
        };
      });
    };

    const getSettingForm = async () => {
      const res = await get(`${endpoint}/settings/${id.value}`);
      const {data} = res;
      form.value = data;
    };

    onMounted(() => {
      getSettingForm();
      getTriggerList();
    });

    const onTabSelect = (tabName) => {
      activeKey.value = tabName;
    };

    const onTriggersChange = (value) => {
      form.value.triggers = [].concat(value);
    };

    const onTitleChange = (value) => {
      form.value.title = value;
    };

    const onTemplateChange = (value) => {
      form.value.template = value;
    };

    return {
      activeKey,
      tabs,
      onBack,
      onSave,
      form,
      formRef,
      triggerList,
      onTabSelect,
      onTriggersChange,
      onTitleChange,
      onTemplateChange,
      t,
    };
  },
});
</script>

<style scoped>
.detail-layout {
  display: flex;
  height: 100%;
}

.detail-layout .content {
  flex: 1;
  max-width: 100%;
  background-color: white;
  display: flex;
  flex-direction: column;
}

.detail-layout .content .content-container {
  margin: 20px;
}
</style>
