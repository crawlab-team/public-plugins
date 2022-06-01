<template>
  <cl-dialog
      :title="t('actions.uninstall')"
      :visible="visible"
      width="640px"
      :confirm-loading="loading"
      @confirm="onConfirm"
      @close="onClose"
  >
    <cl-form>
      <cl-form-item :span="4" :label="t('components.form.dependencyName')">
        <cl-tag
            v-for="n in names"
            :key="n"
            class="dep-name"
            type="primary"
            :label="n"
            size="small"
        />
      </cl-form-item>
      <cl-form-item :span="4" :label="t('components.form.mode')">
        <el-select v-model="mode">
          <el-option value="all" :label="t('components.form.allNodes')"/>
          <el-option value="selected-nodes" :label="t('components.form.selectedNodes')"/>
        </el-select>
      </cl-form-item>
      <cl-form-item v-if="mode === 'selected-nodes'" :span="4" :label="t('components.form.selectedNodes')">
        <el-select v-model="nodeIds" multiple :placeholder="t('components.form.selectNodes')">
          <el-option v-for="n in nodes" :key="n.key" :value="n._id" :label="n.name"/>
        </el-select>
      </cl-form-item>
    </cl-form>
  </cl-dialog>
</template>

<script lang="ts">
import {defineComponent, ref} from 'vue';

const pluginName = 'dependency';
const t = (path) => window['_tp'](pluginName, path);

export default defineComponent({
  name: 'UninstallForm',
  props: {
    visible: {
      type: Boolean,
    },
    names: {
      type: Array,
      default: () => {
        return [];
      },
    },
    nodes: {
      type: Array,
      default: () => {
        return [];
      }
    },
    loading: {
      type: Boolean,
    },
  },
  emits: [
    'confirm',
    'close',
  ],
  setup(props, {emit}) {
    const mode = ref('all');
    const nodeIds = ref([]);

    const reset = () => {
      mode.value = 'all';
      nodeIds.value = [];
    };

    const onConfirm = () => {
      emit('confirm', {
        mode: mode.value,
        nodeIds: nodeIds.value,
      });
      reset();
    };

    const onClose = () => {
      emit('close');
      reset();
    };

    return {
      nodeIds,
      mode,
      onConfirm,
      onClose,
      t,
    };
  },
});
</script>

<style scoped>
.dep-name {
  margin-right: 10px;
}
</style>
